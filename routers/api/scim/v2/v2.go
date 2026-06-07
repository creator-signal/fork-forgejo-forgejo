package v2

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"

	"forgejo.org/models/auth"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	oauth2source "forgejo.org/services/auth/source/oauth2"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
)

// upsertMarkerKey is a context key used to signal that a SCIM Create
// resolved to an existing user (upsert path) so the response writer can
// send HTTP 200 instead of the library's hardcoded 201.
type upsertMarkerKey struct{}

// scimResponseWriter intercepts WriteHeader to downgrade 201 -> 200 when the
// request context indicates an upsert of an already-existing resource.
type scimResponseWriter struct {
	http.ResponseWriter
	ctx context.Context
}

func (rw *scimResponseWriter) WriteHeader(code int) {
	if code == http.StatusCreated {
		if flag, ok := rw.ctx.Value(upsertMarkerKey{}).(*bool); ok && *flag {
			code = http.StatusOK
		}
	}
	rw.ResponseWriter.WriteHeader(code)
}

var scimServerCache sync.Map // key: int64 sourceID -> scim.Server

// Expected URL: /api/scim/{providerName}/v2/
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const mountPrefix = "/api/scim/"

		// r.URL.Path is already URL decoded
		rest := strings.TrimPrefix(r.URL.Path, mountPrefix)
		providerName, _, _ := strings.Cut(rest, "/")
		if providerName == "" {
			http.NotFound(w, r)
			return
		}

		// Derive the percent-encoded segment for correct RawPath stripping.
		providerEncoded := providerName
		if r.URL.RawPath != "" {
			rawRest := strings.TrimPrefix(r.URL.RawPath, mountPrefix)
			providerEncoded, _, _ = strings.Cut(rawRest, "/")
		}

		source, err := auth.GetActiveOAuth2SourceByName(r.Context(), providerName)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		cfg, ok := source.Cfg.(*oauth2source.Source)
		if !ok || !cfg.ScimEnabled {
			http.NotFound(w, r)
			return
		}

		// Validate Bearer token.
		authHeader := r.Header.Get("Authorization")
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := authHeader[len(bearerPrefix):]
		if cfg.ScimToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.ScimToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		server, serverErr := cachedScimServer(source.ID, providerEncoded)
		if serverErr != nil {
			log.Error("SCIM: failed to create server for provider %q: %v", providerName, serverErr)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Strip /api/scim/{providerName} so the library sees /v2/...
		cloned := r.Clone(r.Context())
		cloned.URL.Path = strings.TrimPrefix(r.URL.Path, mountPrefix+providerName)
		if r.URL.RawPath != "" {
			cloned.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, mountPrefix+providerEncoded)
		}

		// Inject an upsert marker into the context so Create() can signal
		// that an existing resource was found; the response writer uses it
		// to return 200 instead of the library's hardcoded 201.
		upsertFlag := new(bool)
		cloned = cloned.WithContext(context.WithValue(cloned.Context(), upsertMarkerKey{}, upsertFlag))

		rw := &scimResponseWriter{ResponseWriter: w, ctx: cloned.Context()}
		server.ServeHTTP(rw, cloned)
	}
}

// cachedScimServer returns a cached scim.Server for the given source, building
// one on first access. The server config is immutable per source so caching is safe.
func cachedScimServer(sourceID int64, providerEncoded string) (scim.Server, error) {
	if cached, ok := scimServerCache.Load(sourceID); ok {
		return cached.(scim.Server), nil
	}
	server, err := newScimServer(sourceID, providerEncoded)
	if err != nil {
		return scim.Server{}, err
	}
	actual, _ := scimServerCache.LoadOrStore(sourceID, server)
	return actual.(scim.Server), nil
}

func newScimServer(sourceID int64, providerEncoded string) (scim.Server, error) {
	absBase := strings.TrimRight(setting.AppURL, "/") + "/api/scim/" + providerEncoded + "/v2"

	config := scim.ServiceProviderConfig{
		DocumentationURI: optional.NewString("https://docs.forgejo.org"),
		SupportFiltering: false,
		SupportPatch:     false,
		MaxResults:       200,
		AuthenticationSchemes: []scim.AuthenticationScheme{{
			DocumentationURI: optional.NewString("https://docs.forgejo.org"),
			Type:             scim.AuthenticationTypeOauthBearerToken,
			Name:             "OAuth Bearer Token",
			Description:      "Authentication via a static Bearer token configured per authentication source",
			SpecURI:          optional.NewString("http://www.rfc-editor.org/info/rfc6750"),
			Primary:          true,
		}},
	}

	userSchema := schema.Schema{
		ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
		Name:        optional.NewString("User"),
		Description: optional.NewString("User Account"),
		Attributes: []schema.CoreAttribute{
			schema.SimpleCoreAttribute(schema.SimpleStringParams(schema.StringParams{
				Name:       "userName",
				Required:   true,
				Uniqueness: schema.AttributeUniquenessServer(),
			})),
			schema.SimpleCoreAttribute(schema.SimpleStringParams(schema.StringParams{
				Description: optional.NewString("The name of the User, suitable for display to end-users."),
				Name:        "displayName",
			})),
			schema.ComplexCoreAttribute(schema.ComplexParams{
				Description: optional.NewString("Email addresses for the user."),
				MultiValued: true,
				Name:        "emails",
				SubAttributes: []schema.SimpleParams{
					schema.SimpleStringParams(schema.StringParams{
						Name: "value",
					}),
					schema.SimpleBooleanParams(schema.BooleanParams{
						Description: optional.NewString("Indicates the preferred email address."),
						Name:        "primary",
					}),
				},
			}),
			schema.SimpleCoreAttribute(schema.SimpleBooleanParams(schema.BooleanParams{
				Description: optional.NewString("A Boolean value indicating the User's administrative status."),
				Name:        "active",
			})),
		},
	}

	return scim.NewServer(
		&scim.ServerArgs{
			ServiceProviderConfig: &config,
			ResourceTypes: []scim.ResourceType{
				{
					ID:               optional.NewString("User"),
					Name:             "User",
					Endpoint:         "/Users",
					Description:      optional.NewString("User Account"),
					Schema:           userSchema,
					SchemaExtensions: []scim.SchemaExtension{},
					Handler:          userResourceHandler{sourceID: sourceID},
				},
			},
		},
		scim.WithBaseURL(absBase),
	)
}
