package integration

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	opentofu_model "forgejo.org/models/packages/opentofu"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	opentofu_state_module "forgejo.org/modules/packages/opentofu/state"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageOpenTofuHttpBackend(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	rootURL := fmt.Sprintf("/api/packages/%s/opentofu/http/state", user.Name)

	unencryptedVersion4StateFile := `{
		"version": 4,
		"terraform_version": "1.9.0",
		"serial": 1,
		"lineage": "fcf2c3a0-3a0f-703f-c611-0ca7776c06b8",
		"outputs": {},
		"resources": [],
		"check_results": null
	}`
	encryptedVersion4StateFile := `{
		"serial": 1,
		"lineage": "2ce64317-825a-f8cd-b35e-84fa110b561f",
		"meta": {
			"key_provider.pbkdf2.local": "eyJzYWx0IjoiNW9WSGpZM3BGaHRaNncwUE1hSDkrcnc2UzR1cjE4Q05adUR4RXNWM1dmcz0iLCJpdGVyYXRpb25zIjo2MDAwMDAsImhhc2hfZnVuY3Rpb24iOiJzaGE1MTIiLCJrZXlfbGVuZ3RoIjozMn0="
		},
		"encrypted_data": "e9oQmRWzPOcELCgmi1uJd5vfKMSnO7oQe/Mpyo8pKEXyNwChoPs0HpDeFhf2h32diyUORX28rWQ+D+wotHseJ+9c6EeE/G3TNr+ilmvdy0nKqZ4ABw/YoD6Zwn+DKF4qviUK2pnXN3HOQqgHUMmiuzL/AqIz1gDBncDyKJLdzuAXjXI1NyFRtCNJmVJpJkyq0ZhQG/i1KxbxAQtCAySH8T3KPE/njy4X7E08vy29rHtGxF0=",
		"encryption_version": "v0"
	}`

	lockRequestPayload := `{
		"ID": "55769533-286a-96d9-eb47-935c08f675a9",
		"Operation": "OperationTypeApply",
		"Info": "",
		"Who": "user@laptop",
		"Version": "1.10.6",
		"Created": "2025-12-01T16:37:39.015107776Z",
		"Path": ""
	}`

	t.Run("Lock", func(t *testing.T) {
		// Sends a lock request without being authenticated.
		t.Run("UnauthenticatedLockRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/unauthenticated/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json")
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		// Sends a lock request with an invalid JSON payload.
		t.Run("InvalidJSON", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/invalid-json/lock", strings.NewReader("This is not a valid JSON payload")).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusBadRequest)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid lock request.
		t.Run("ValidLockRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"

			lock, err := opentofu_model.GetLock(db.DefaultContext, packageName, user.ID)
			assert.Nil(t, lock)
			require.ErrorIs(t, err, opentofu_model.ErrStateLockNotExist{PackageName: packageName, OwnerID: user.ID})

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/"+packageName+"/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")

			lock, err = opentofu_model.GetLock(db.DefaultContext, packageName, user.ID)
			require.NoError(t, err)
			assert.Equal(t, "55769533-286a-96d9-eb47-935c08f675a9", lock.LockID)
			assert.Equal(t, "OperationTypeApply", lock.Operation)
			assert.Equal(t, user.Name, lock.UserName)
			assert.Equal(t, "1.10.6", lock.ClientVersion)
		})
	})

	t.Run("Upload", func(t *testing.T) {
		// Sends a state upload request without being authenticated.
		t.Run("UnauthenticatedUploadRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/unauthenticated", strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json")
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		// Sends a state upload request with an invalid JSON payload.
		t.Run("InvalidJSON", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/invalid-json", strings.NewReader("This is not a valid JSON payload")).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusBadRequest)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a state upload request with an invalid MD5 checksum as HTTP header.
		t.Run("InvalidMD5Checksum", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/invalid-md5", strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", "SW52YWxpZCBtZDUgY2hlY2tzdW0K").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusBadRequest)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unencrypted version 4 state file but without the lock ID while
		// the state file is locked.
		//
		// The state file has been locked in a previous test.
		t.Run("MissingLockID", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/"+packageName, strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusConflict)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unencrypted version 4 state file but with an invalid lock ID
		// while the state file is locked.
		//
		// The state file has been locked in a previous test.
		t.Run("InvalidLockID", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"
			lockID := "wrong-lock-id"

			req := NewRequestWithBody(t, http.MethodPost, fmt.Sprintf("%s/%s?ID=%s", rootURL, packageName, lockID), strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusUnauthorized)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unencrypted version 4 state file with the correct lock ID as
		// the state file has been locked in a previous test.
		t.Run("UploadValidUnencryptedVersion4StateFile", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"
			packageVersion := "1"
			lockID := "55769533-286a-96d9-eb47-935c08f675a9"

			pvs, err := packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Empty(t, pvs)

			md5Hash := md5.Sum([]byte(unencryptedVersion4StateFile))
			md5Base64 := base64.StdEncoding.EncodeToString(md5Hash[:])

			req := NewRequestWithBody(t, http.MethodPost, fmt.Sprintf("%s/%s?ID=%s", rootURL, packageName, lockID), strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusCreated)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Empty(t, bodyBytes)

			pvs, err = packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Len(t, pvs, 1)

			pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
			require.NoError(t, err)
			assert.Equal(t, packageName, pd.Package.Name)
			assert.Equal(t, packageVersion, pd.Version.Version)
			assert.IsType(t, &opentofu_state_module.Metadata{}, pd.Metadata)
			assert.False(t, pd.Metadata.(*opentofu_state_module.Metadata).Encrypted)

			pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
			require.NoError(t, err)
			assert.Len(t, pfs, 1)
			assert.Equal(t, opentofu_state_module.OpenTofuStateFilename, pfs[0].Name)
			assert.True(t, pfs[0].IsLead)

			pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
			require.NoError(t, err)
			assert.Equal(t, int64(len(unencryptedVersion4StateFile)), pb.Size)
			assert.Equal(t, pd.Metadata.(*opentofu_state_module.Metadata).ChecksumMD5, md5.Sum([]byte(unencryptedVersion4StateFile)))

			req = NewRequestWithBody(t, http.MethodPost, rootURL+"/"+packageName, strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			resp = MakeRequest(t, req, http.StatusConflict)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid encrypted version 4 state file.
		t.Run("UploadValidEncryptedVersion4StateFile", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-encrypted"
			packageVersion := "1"

			pvs, err := packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Empty(t, pvs)

			md5Hash := md5.Sum([]byte(encryptedVersion4StateFile))
			md5Base64 := base64.StdEncoding.EncodeToString(md5Hash[:])

			req := NewRequestWithBody(t, http.MethodPost, rootURL+"/"+packageName, strings.NewReader(encryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusCreated)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Empty(t, bodyBytes)

			pvs, err = packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Len(t, pvs, 1)

			pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
			require.NoError(t, err)
			assert.Equal(t, packageName, pd.Package.Name)
			assert.Equal(t, packageVersion, pd.Version.Version)
			assert.IsType(t, &opentofu_state_module.Metadata{}, pd.Metadata)
			assert.True(t, pd.Metadata.(*opentofu_state_module.Metadata).Encrypted)

			pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
			require.NoError(t, err)
			assert.Len(t, pfs, 1)
			assert.Equal(t, opentofu_state_module.OpenTofuStateFilename, pfs[0].Name)
			assert.True(t, pfs[0].IsLead)

			pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
			require.NoError(t, err)
			assert.Equal(t, int64(len(encryptedVersion4StateFile)), pb.Size)
			assert.Equal(t, pd.Metadata.(*opentofu_state_module.Metadata).ChecksumMD5, md5.Sum([]byte(encryptedVersion4StateFile)))

			req = NewRequestWithBody(t, http.MethodPost, rootURL+"/"+packageName, strings.NewReader(encryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			resp = MakeRequest(t, req, http.StatusConflict)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})
	})

	t.Run("Fetch", func(t *testing.T) {
		// Sends a fetch request for a non-existing package.
		t.Run("NoStateFile", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, http.MethodGet, rootURL+"/non-existing")
			MakeRequest(t, req, http.StatusNoContent)
		})

		// Sends a fetch request to download the previously uploaded 'v4-unencrypted'
		// package.
		t.Run("FetchUnencryptedVersion4Package", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, http.MethodGet, rootURL+"/v4-unencrypted")
			resp := MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NotEmpty(t, bodyBytes)
			assert.Equal(t, unencryptedVersion4StateFile, string(bodyBytes))
		})
	})

	t.Run("Delete", func(t *testing.T) {
		// Sends a state deletion request without being authenticated.
		t.Run("UnauthenticatedDeletionRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, http.MethodDelete, rootURL+"/unauthenticated")
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		// Sends a deletion request for a non-existing package.
		t.Run("NoStateFile", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, http.MethodDelete, rootURL+"/non-existing").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusNotFound)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a deletion request to delete all versions of the previously uploaded
		// 'v4-encrypted' package.
		t.Run("DeleteEncryptedVersion4Package", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-encrypted"

			req := NewRequest(t, http.MethodDelete, rootURL+"/"+packageName).AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusOK)

			pvs, err := packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Empty(t, pvs)
		})
	})

	t.Run("Unlock", func(t *testing.T) {
		// Sends an unlock request without being authenticated.
		t.Run("UnauthenticatedUnlockRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodDelete, rootURL+"/unauthenticated/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json")
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		// Sends an unlock request with an invalid JSON payload.
		t.Run("InvalidJSON", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodDelete, rootURL+"/invalid-json/lock", strings.NewReader("This is not a valid JSON payload")).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusBadRequest)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unlock request for an unknown package/state file.
		t.Run("UnknownPackage", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, http.MethodDelete, rootURL+"/unknown-package/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusNotFound)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unlock request for an already unlocked package/state file.
		t.Run("AlreadyUnlockedPackage", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-encrypted"

			req := NewRequestWithBody(t, http.MethodDelete, rootURL+"/"+packageName+"/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusNotFound)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unlock request but with an invalid lock ID.
		//
		// The state file has been locked in a previous test.
		t.Run("InvalidLockID", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"
			lockRequestPayload := `{
				"ID": "invalid-lock-id",
				"Operation": "OperationTypeApply",
				"Info": "",
				"Who": "user@laptop",
				"Version": "1.10.6",
				"Created": "2025-12-01T16:37:39.015107776Z",
				"Path": ""
			}`

			req := NewRequestWithBody(t, http.MethodDelete, rootURL+"/"+packageName+"/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusUnauthorized)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
		})

		// Sends a valid unlock request.
		//
		// The state file has been locked in a previous test.
		t.Run("ValidUnlockRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"

			_, err := opentofu_model.GetLock(db.DefaultContext, packageName, user.ID)
			require.NoError(t, err)

			req := NewRequestWithBody(t, http.MethodDelete, rootURL+"/"+packageName+"/lock", strings.NewReader(lockRequestPayload)).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusOK)

			lock, err := opentofu_model.GetLock(db.DefaultContext, packageName, user.ID)
			assert.Nil(t, lock)
			require.ErrorIs(t, err, opentofu_model.ErrStateLockNotExist{PackageName: packageName, OwnerID: user.ID})
		})
	})
}
