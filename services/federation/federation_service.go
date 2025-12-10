// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"forgejo.org/models/federation_key"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/auth/password"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/validation"

	"github.com/google/uuid"
)

func Init() error {
	if !setting.Federation.Enabled {
		return nil
	}
	return initDeliveryQueue()
}

func FindOrCreateFederationHost(ctx context.Context, actorURI string) (*forgefed.FederationHost, error) {
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := forgefed.FindFederationHostByFqdnAndPort(ctx, rawActorID.Host, rawActorID.HostPort)
	if err != nil {
		return nil, err
	}
	if !federationHost.Has() {
		result, err := createFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = optional.Some(result)
	}
	return federationHost.Value(), nil
}

func FindOrCreateFederatedUser(ctx context.Context, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	user, federatedUser, federationHost, err := findFederatedUser(ctx, actorURI)
	if err != nil {
		return nil, nil, nil, err
	}
	personID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		return nil, nil, nil, err
	}

	if user != nil {
		log.Trace("Local ActivityPub user found (actorURI: %#v, user: %#v)", actorURI, user)
	} else {
		log.Trace("Attempting to create new user and federatedUser for actorURI: %#v", actorURI)
		user, federatedUser, err = createUserFromAP(ctx, personID, federationHost.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		log.Trace("Created user %#v with federatedUser %#v from distant server", user, federatedUser)
	}
	log.Trace("Got user: %v", user.Name)

	return user, federatedUser, federationHost, nil
}

func findFederatedUser(ctx context.Context, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	federationHost, err := FindOrCreateFederationHost(ctx, actorURI)
	if err != nil {
		return nil, nil, nil, err
	}
	actorID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		return nil, nil, nil, err
	}

	user, federatedUser, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	return user.Value(), federatedUser.Value(), federationHost, nil
}

func createFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*forgefed.FederationHost, error) {
	actionsUser := user.NewAPServerActor()

	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, err
	}

	client, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.KeyID())
	if err != nil {
		return nil, err
	}

	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return nil, err
	}

	nodeInfoWellKnown, err := forgefed.NewNodeInfoWellKnown(body)
	if err != nil {
		return nil, err
	}

	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return nil, err
	}

	nodeInfo, err := forgefed.NewNodeInfo(body)
	if err != nil {
		return nil, err
	}

	// TODO: we should get key material here also to have it immediately
	result, err := forgefed.NewFederationHost(actorID.Host, nodeInfo, actorID.HostPort, actorID.HostSchema)
	if err != nil {
		return nil, err
	}

	err = forgefed.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func fetchUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, *federation_key.FederationPublicKey, error) {
	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.KeyID())
	if err != nil {
		return nil, nil, nil, err
	}

	body, err := apClient.GetBody(personID.AsURI())
	if err != nil {
		return nil, nil, nil, err
	}

	person := fm.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return nil, nil, nil, err
	}

	if res, err := validation.IsValid(person); !res {
		return nil, nil, nil, err
	}

	log.Info("Fetched valid person from distant server: %q", person)

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return nil, nil, nil, err
	}

	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	fullName := person.Name.String()

	if len(person.Name) == 0 {
		fullName = name
	}

	password, err := password.Generate(32)
	if err != nil {
		return nil, nil, nil, err
	}

	inbox, err := url.ParseRequestURI(person.Inbox.GetLink().String())
	if err != nil {
		return nil, nil, nil, err
	}

	pubKeyBytes, err := decodePublicKeyPem(person.PublicKey.PublicKeyPem)
	if err != nil {
		return nil, nil, nil, err
	}

	newUser := user.User{
		LowerName:                    strings.ToLower(name),
		Name:                         name,
		FullName:                     fullName,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    loginName,
		Type:                         user.UserTypeRemoteUser,
		IsAdmin:                      false,
	}

	federationPublicKey, err := federation_key.NewFederationPublicKey(0, person.PublicKey.ID.String(), pubKeyBytes, 0, federation_key.FederatedUserType, federation_key.RsaSha256Cavage)
	if err != nil {
		return nil, nil, nil, err
	}

	federatedUser := user.FederatedUser{
		ExternalID:            personID.ID,
		FederationHostID:      federationHostID,
		InboxPath:             inbox.Path,
		NormalizedOriginalURL: personID.AsURI(),
	}

	if err = federatedUser.ValidateKeyID(ctx, federationPublicKey.KeyID); err != nil {
		return nil, nil, nil, err
	}

	log.Info("Fetched person's %q federatedUser from distant server: %q", person, federatedUser)
	return &newUser, &federatedUser, federationPublicKey, nil
}

func createUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	newUser, federatedUser, federationPublicKey, err := fetchUserFromAP(ctx, personID, federationHostID)
	if err != nil {
		return nil, nil, err
	}
	err = user.CreateFederatedUser(ctx, newUser, federatedUser)
	if err != nil {
		return nil, nil, err
	}

	_, optFederatedUser, err := user.FindFederatedUser(ctx, federatedUser.ExternalID, federatedUser.FederationHostID)
	if err != nil {
		return nil, nil, err
	} else if optFederatedUser == nil {
		return nil, nil, fmt.Errorf("missing federated user: %v", federatedUser.ExternalID)
	}
	federatedUser = optFederatedUser.Value()

	federationPublicKey.ActorID = federatedUser.ID

	if _, err = federation_key.FindOrCreateFederationPublicKey(ctx, federationPublicKey); err != nil {
		return nil, nil, err
	}

	log.Info("Created federatedUser: %q", federatedUser)
	return newUser, federatedUser, nil
}
