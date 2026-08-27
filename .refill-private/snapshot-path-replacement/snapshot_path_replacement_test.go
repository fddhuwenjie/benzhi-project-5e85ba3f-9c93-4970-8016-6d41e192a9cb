package snapshot_path_replacement_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func TestCommitFollowsReplacedSnapshotPath(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "release.json")
	store, err := repository.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := application.New(store, audit.New(func() time.Time { return now }), func() time.Time { return now })

	created, err := service.Create(ctx, application.CreateInput{
		CommandMeta: application.CommandMeta{RequestID: "request-create-path-replacement", Actor: "负责人甲", Role: domain.RoleOwner},
		ID:          "release-path-replacement", Title: "路径轮转试验", Objective: "验证快照替换后的提交可恢复", ModelCode: "MODEL-PATH", PlannedCondition: "常温工况", Owner: "负责人甲",
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	baseline, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read baseline snapshot: %v", err)
	}
	if err := os.Rename(path, path+".rotated"); err != nil {
		t.Fatalf("rotate snapshot: %v", err)
	}
	if err := os.WriteFile(path, baseline, 0600); err != nil {
		t.Fatalf("install replacement snapshot: %v", err)
	}

	updated, err := service.UpdateProfile(ctx, created.Release.ID, application.ProfileInput{
		CommandMeta: application.CommandMeta{RequestID: "request-update-path-replacement", ExpectedRevision: created.Release.Revision, Actor: "负责人甲", Role: domain.RoleOwner},
		Title:       "路径轮转后已更新", Objective: created.Release.Objective, ModelCode: created.Release.ModelCode, PlannedCondition: created.Release.PlannedCondition, Owner: created.Release.Owner, ConfirmDiff: true,
	})
	if err != nil {
		t.Fatalf("update after replacement: %v", err)
	}
	if updated.Release.Revision != 2 {
		t.Fatalf("live service revision = %d, want 2", updated.Release.Revision)
	}

	reopened, err := repository.Open(path)
	if err != nil {
		t.Fatalf("reopen replacement path: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	recovered, err := reopened.Load(ctx, created.Release.ID)
	if err != nil {
		t.Fatalf("load recovered release: %v", err)
	}
	if recovered.Revision != updated.Release.Revision || recovered.Title != updated.Release.Title {
		t.Fatalf("TestCommitFollowsReplacedSnapshotPath: recovered revision/title = %d/%q, want %d/%q", recovered.Revision, recovered.Title, updated.Release.Revision, updated.Release.Title)
	}
}
