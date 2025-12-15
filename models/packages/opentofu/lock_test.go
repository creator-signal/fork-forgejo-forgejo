package opentofu

import (
	"fmt"
	"strings"
	"testing"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/packages"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	packages_service "forgejo.org/services/packages"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func preparePackage(t *testing.T, owner *user_model.User, name string) int64 {
	t.Helper()

	data, err := packages.CreateHashedBufferFromReader(strings.NewReader("data"))
	require.NoError(t, err)

	pv, _, err := packages_service.CreatePackageOrAddFileToExisting(
		db.DefaultContext,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       owner,
				PackageType: packages_model.TypeOpenTofuState,
				Name:        name,
			},
			Creator: owner,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: name,
			},
			Data:    data,
			Creator: owner,
			IsLead:  true,
		},
	)

	require.NoError(t, err)
	return pv.PackageID
}

func prepareStateLock(t *testing.T, lockInfo *StateLock) {
	t.Helper()

	e := db.GetEngine(db.DefaultContext)

	_, err := e.Insert(lockInfo)
	require.NoError(t, err)
}

func TestLockUniqueness(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()

	packageName := "opentofu-state"

	// Try to insert twice the same combination of package name and owner ID that
	// must be unique.
	t.Run("InsertTwice", func(t *testing.T) {
		e := db.GetEngine(db.DefaultContext)

		_, err := e.Insert(StateLock{
			PackageName: packageName,
			OwnerID:     1,
		})
		require.NoError(t, err)

		_, err = e.Insert(StateLock{
			PackageName: packageName,
			OwnerID:     2,
		})
		require.NoError(t, err)

		_, err = e.Insert(StateLock{
			PackageName: packageName,
			OwnerID:     1,
		})
		require.Error(t, err)
	})
}

func TestGetLock(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	clientVersion := "OpenTofu v1.10.6"
	extraInfo := "Dummy extra info"

	packageNameUnlocked := "opentofu-state-unlocked"
	preparePackage(t, user, packageNameUnlocked)

	packageNameLocked := "opentofu-state-locked"
	preparePackage(t, user, packageNameLocked)
	lockIDLocked := "110e9066-29ca-49b6-b507-87c58bd1177b"
	prepareStateLock(t, &StateLock{
		PackageName:   packageNameLocked,
		OwnerID:       user.ID,
		LockID:        lockIDLocked,
		Info:          extraInfo,
		UserName:      user.Name,
		ClientVersion: clientVersion,
	})

	// Try to get the state lock of a non-existing package.
	t.Run("NoPackage", func(t *testing.T) {
		lock, err := GetLock(db.DefaultContext, "non-existing-package", user.ID)
		require.Nil(t, lock)
		require.Error(t, err)
		require.IsType(t, ErrStateLockNotExist{}, err)
		expectedError := fmt.Sprintf("package %s is not locked [OwnerID: %d]", "non-existing-package", user.ID)
		require.Errorf(t, err, expectedError)
	})

	// Try to get the state lock of an existing package that is not locked.
	t.Run("NoStateLock", func(t *testing.T) {
		lock, err := GetLock(db.DefaultContext, packageNameUnlocked, user.ID)
		require.Nil(t, lock)
		require.Error(t, err)
		require.IsType(t, ErrStateLockNotExist{}, err)
		expectedError := fmt.Sprintf("package %s is not locked [OwnerID: %d]", packageNameUnlocked, user.ID)
		require.Errorf(t, err, expectedError)
	})

	// Get the state lock from the database.
	t.Run("GetStateLock", func(t *testing.T) {
		lock, err := GetLock(db.DefaultContext, packageNameLocked, user.ID)
		require.NoError(t, err)
		assert.Equal(t, packageNameLocked, lock.PackageName)
		assert.Equal(t, user.ID, lock.OwnerID)
		assert.Equal(t, lockIDLocked, lock.LockID)
		assert.Equal(t, extraInfo, lock.Info)
		assert.Equal(t, user.Name, lock.UserName)
		assert.Equal(t, clientVersion, lock.ClientVersion)
	})
}

func TestLock(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageNameNonExisting := "non-existing-package"
	clientVersion := "OpenTofu v1.10.6"
	extraInfo := "Dummy extra info"

	packageNameUnlocked := "opentofu-state-unlocked"
	preparePackage(t, user, packageNameUnlocked)
	lockIDUnlocked := "f0f34926-f710-44ec-9fe0-1605603ebf4b"

	packageNameLocked := "opentofu-state-locked"
	preparePackage(t, user, packageNameLocked)
	lockIDLocked := "110e9066-29ca-49b6-b507-87c58bd1177b"
	prepareStateLock(t, &StateLock{
		PackageName:   packageNameLocked,
		OwnerID:       user.ID,
		LockID:        lockIDLocked,
		Info:          extraInfo,
		UserName:      user.Name,
		ClientVersion: clientVersion,
	})

	// Try to lock a non-existing package.
	//
	// Locking a non-existing package should not produce any error as both OpenTofu
	// and Terraform send a lock request when uploading a state file for the very
	// first time.
	t.Run("NoPackage", func(t *testing.T) {
		lock, err := Lock(db.DefaultContext, &StateLock{
			PackageName:   packageNameNonExisting,
			OwnerID:       user.ID,
			LockID:        lockIDUnlocked,
			Info:          extraInfo,
			UserName:      user.Name,
			ClientVersion: clientVersion,
		})
		require.NoError(t, err)
		assert.Equal(t, packageNameNonExisting, lock.PackageName)
		assert.Equal(t, user.ID, lock.OwnerID)
		assert.Equal(t, lockIDUnlocked, lock.LockID)
		assert.Equal(t, extraInfo, lock.Info)
		assert.Equal(t, user.Name, lock.UserName)
		assert.Equal(t, clientVersion, lock.ClientVersion)
	})

	// Try to lock an already-locked package.
	t.Run("AlreadyLocked", func(t *testing.T) {
		lock, err := Lock(db.DefaultContext, &StateLock{
			PackageName: packageNameLocked,
			OwnerID:     user.ID,
			LockID:      "dummy-lock-id",
		})
		require.Error(t, err)
		require.IsType(t, ErrStateLockAlreadyExist{}, err)
		expectedError := fmt.Sprintf("package %s is already locked [OwnerID: %d]", packageNameLocked, user.ID)
		require.Errorf(t, err, expectedError)

		assert.Equal(t, packageNameLocked, lock.PackageName)
		assert.Equal(t, user.ID, lock.OwnerID)
		assert.Equal(t, lockIDLocked, lock.LockID)
		assert.Equal(t, extraInfo, lock.Info)
		assert.Equal(t, user.Name, lock.UserName)
		assert.Equal(t, clientVersion, lock.ClientVersion)
	})

	// Lock an OpenTofu/Terraform state file/package.
	t.Run("LockState", func(t *testing.T) {
		lock, err := Lock(db.DefaultContext, &StateLock{
			PackageName:   packageNameUnlocked,
			OwnerID:       user.ID,
			LockID:        lockIDUnlocked,
			Info:          extraInfo,
			UserName:      user.Name,
			ClientVersion: clientVersion,
		})
		require.NoError(t, err)
		assert.Equal(t, packageNameUnlocked, lock.PackageName)
		assert.Equal(t, user.ID, lock.OwnerID)
		assert.Equal(t, lockIDUnlocked, lock.LockID)
		assert.Equal(t, extraInfo, lock.Info)
		assert.Equal(t, user.Name, lock.UserName)
		assert.Equal(t, clientVersion, lock.ClientVersion)
	})
}

func TestUnlock(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageNameUnlocked := "opentofu-state-unlocked"
	preparePackage(t, user, packageNameUnlocked)
	lockIDUnlocked := "f0f34926-f710-44ec-9fe0-1605603ebf4b"

	packageNameLocked := "opentofu-state-locked"
	preparePackage(t, user, packageNameLocked)
	lockIDLocked := "110e9066-29ca-49b6-b507-87c58bd1177b"
	prepareStateLock(t, &StateLock{
		PackageName: packageNameLocked,
		OwnerID:     user.ID,
		LockID:      lockIDLocked,
	})

	// Try to unlock of a non-existing package.
	t.Run("NoPackage", func(t *testing.T) {
		err := Unlock(db.DefaultContext, "non-existing-package", user.ID, lockIDLocked)
		require.Error(t, err)
		require.IsType(t, ErrStateLockNotExist{}, err)
		expectedError := fmt.Sprintf("package %s is not locked [OwnerID: %d]", "non-existing-package", user.ID)
		require.Errorf(t, err, expectedError)
	})

	// Try to unlock an already unlocked package.
	t.Run("NotLocked", func(t *testing.T) {
		err := Unlock(db.DefaultContext, packageNameUnlocked, user.ID, lockIDUnlocked)
		require.Error(t, err)
		require.IsType(t, ErrStateLockNotExist{}, err)
		expectedError := fmt.Sprintf("package %s is not locked [OwnerID: %d]", packageNameUnlocked, user.ID)
		require.Errorf(t, err, expectedError)
	})

	// Try to unlock a package with an invalid lock ID.
	t.Run("InvalidLockID", func(t *testing.T) {
		err := Unlock(db.DefaultContext, packageNameLocked, user.ID, "invalid-lock-id")
		require.Error(t, err)
		require.IsType(t, ErrInvalidLockID{}, err)
		expectedError := fmt.Sprintf("lock ID %s is invalid [PackageName: %s, OwnerID: %d]", "invalid-lock-id", packageNameLocked, user.ID)
		require.Errorf(t, err, expectedError)
	})

	// Unlock an OpenTofu/Terraform state file/package.
	t.Run("UnlockState", func(t *testing.T) {
		err := Unlock(db.DefaultContext, packageNameLocked, user.ID, lockIDLocked)
		require.NoError(t, err)
	})
}
