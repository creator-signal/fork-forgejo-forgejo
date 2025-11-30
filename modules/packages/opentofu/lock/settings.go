package lock

const (
	// The maximum size of the lock information payload sent by a client in a lock
	// request.
	//
	// This limit is used to avoid denial of service attacks.
	LimitSizeLockInfo int64 = 2000
)
