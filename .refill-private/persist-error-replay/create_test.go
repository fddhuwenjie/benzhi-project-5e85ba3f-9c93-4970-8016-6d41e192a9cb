package persisterrorreplay

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func TestCreateDoesNotReplayAfterPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	store, err := repository.Open(filepath.Join(root, "missing", "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	app := application.New(store, audit.New(func() time.Time { return time.Unix(0, 0) }), func() time.Time { return time.Unix(0, 0) })
	_, err = app.Create(t.Context(), application.CreateInput{
		CommandMeta: application.CommandMeta{RequestID: "persist-failure-001", Actor: "负责人", Role: "owner"},
		ID:          "persist-release", Title: "试验", Objective: "目标", ModelCode: "M-PERSIST", PlannedCondition: "标准", Owner: "负责人",
	})
	if err == nil {
		t.Fatal("持久化失败必须向调用方传播，不能被幂等重放掩盖")
	}
	if _, loadErr := store.Load(t.Context(), "persist-release"); !errors.Is(loadErr, domain.ErrNotFound) {
		t.Fatalf("失败事务不得污染内存状态，Load 返回 %v", loadErr)
	}
}
