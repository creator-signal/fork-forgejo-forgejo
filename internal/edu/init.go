package edu

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"xorm.io/xorm"
)

func Init(ctx context.Context) error {
	log.Info("Initializing Educational Extension...")

	e := db.GetEngine(ctx)
	if e == nil {
		log.Fatal("Educational Extension: Database engine not available")
		return nil
	}
	
	if err := e.Sync(new(Assignment), new(Submission), new(TestResult), new(UserRole)); err != nil {
		log.Error("Educational Extension: Failed to sync database schema: %v", err)
		return err
	}

	var runner SQLRunner
	var xormEngine *xorm.Engine

	if sess, ok := e.(*xorm.Session); ok {
		xormEngine = sess.Engine()
	} else {
		var err error
		xormEngine, err = db.GetMasterEngine(e)
		if err != nil {
			log.Error("Educational Extension: Failed to get master engine: %v", err)
			return nil
		}
	}

	runner = xormEngine.DB().DB

	repo := NewRepository(runner)

	RegisterNotifier(repo)

	log.Info("Educational Extension initialized successfully.")
	return nil
}
