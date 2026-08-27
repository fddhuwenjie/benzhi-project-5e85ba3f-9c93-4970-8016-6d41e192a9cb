package evidence_context_reuse_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

type contextCheckingStore struct {
	data   []byte
	digest string
	calls  int
}

func newContextCheckingStore() *contextCheckingStore {
	data := []byte(`{"schema_version":"wind-tunnel-release-evidence-v1","release_id":"rel-evidence","released_revision":2,"authorization":{"signed_revision":1}}`)
	sum := sha256.Sum256(data)
	return &contextCheckingStore{data: data, digest: hex.EncodeToString(sum[:])}
}

func (s *contextCheckingStore) Evidence(ctx context.Context, _ string) ([]byte, string, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	return append([]byte(nil), s.data...), s.digest, nil
}

func (s *contextCheckingStore) Create(context.Context, repository.Commit) error { return nil }
func (s *contextCheckingStore) Load(context.Context, string) (*domain.TestRelease, error) {
	return nil, domain.ErrNotFound
}
func (s *contextCheckingStore) List(context.Context, int, int) (domain.Page[domain.TestRelease], error) {
	return domain.Page[domain.TestRelease]{}, nil
}
func (s *contextCheckingStore) FindActive(context.Context, string, string, string) ([]repository.ReleaseMatch, error) {
	return nil, nil
}
func (s *contextCheckingStore) Commit(context.Context, repository.Commit) error { return nil }
func (s *contextCheckingStore) FindIdempotency(context.Context, string) (*repository.IdempotencyRecord, error) {
	return nil, domain.ErrNotFound
}
func (s *contextCheckingStore) Audit(context.Context, string, string, int, int) (domain.Page[domain.AuditEvent], error) {
	return domain.Page[domain.AuditEvent]{}, nil
}
func (s *contextCheckingStore) QueryAudit(context.Context, string, repository.AuditFilter) (repository.AuditView, error) {
	return repository.AuditView{}, nil
}
func (s *contextCheckingStore) Close() error { return nil }

func TestEvidenceLoaderUsesCurrentRequestContext(t *testing.T) {
	store := newContextCheckingStore()
	service := application.New(store, audit.New(time.Now), time.Now)
	firstCtx, finishFirstRequest := context.WithCancel(context.Background())

	if _, _, err := service.Evidence(firstCtx, "rel-evidence"); err != nil {
		t.Fatalf("首个证据下载失败: %v", err)
	}
	finishFirstRequest()

	if _, _, err := service.Evidence(context.Background(), "rel-evidence"); err != nil {
		t.Fatalf("后续健康请求复用了已取消 context: %v", err)
	}
	if store.calls != 2 {
		t.Fatalf("仓储读取次数 = %d，期望 2", store.calls)
	}
}
