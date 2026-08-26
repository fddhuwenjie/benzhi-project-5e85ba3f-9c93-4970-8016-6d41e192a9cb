package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/repository"
	"windtunnel-release/internal/web"
)

const defaultAddr = "127.0.0.1:19081"

func main() {
	addrFlag := flag.String("addr", "", "监听地址，仅允许 127.0.0.1:<port>")
	selfcheck := flag.Bool("selfcheck", false, "运行有界 HTTP 自检后退出")
	dbPath := flag.String("db", "", "SQLite 文件路径")
	flag.Parse()
	addr, err := resolveAddr(*addrFlag, os.Getenv("PORT"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *selfcheck && *dbPath == "" {
		*dbPath = ":memory:"
	}
	store, err := repository.Open(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()
	clock := time.Now
	auditService := audit.New(clock)
	app := application.New(store, auditService, clock)
	server := &http.Server{Addr: addr, Handler: web.New(app).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 30 * time.Second}
	if *selfcheck {
		if err := runSelfcheck(server, addr); err != nil {
			fmt.Fprintln(os.Stderr, "selfcheck 失败:", err)
			os.Exit(1)
		}
		fmt.Println("selfcheck 通过")
		return
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "监听失败:", err)
		os.Exit(1)
	}
	fmt.Println("风洞试验安全放行台监听", listener.Addr().String())
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "服务异常:", err)
		os.Exit(1)
	}
}

func resolveAddr(flagAddr, port string) (string, error) {
	if flagAddr != "" {
		return validateLoopback(flagAddr)
	}
	if port != "" {
		if strings.Contains(port, ":") {
			return "", fmt.Errorf("PORT 只接受端口号")
		}
		return validateLoopback("127.0.0.1:" + port)
	}
	return defaultAddr, nil
}

func validateLoopback(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("监听地址格式无效: %w", err)
	}
	if host != "127.0.0.1" {
		return "", fmt.Errorf("仅允许绑定 127.0.0.1")
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1024 || p > 65535 {
		return "", fmt.Errorf("端口必须在 1024 到 65535 之间")
	}
	return net.JoinHostPort(host, port), nil
}

func runSelfcheck(server *http.Server, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server.Addr = listener.Addr().String()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	client := &http.Client{Timeout: 5 * time.Second}
	base := "http://" + listener.Addr().String()
	if err := waitHealth(client, base+"/healthz"); err != nil {
		_ = server.Shutdown(context.Background())
		return err
	}
	id := "selfcheck-release-" + time.Now().UTC().Format("20060102150405.000000000")
	var release map[string]any
	var err2 error
	post := func(path string, body any) map[string]any {
		result, err := doJSON(client, http.MethodPost, base+path, body)
		if err != nil {
			err2 = err
		}
		return result
	}
	release = post("/api/releases", map[string]any{"request_id": "selfcheck-create-001", "expected_revision": 0, "actor": "试验负责人", "role": "owner", "id": id, "title": "自检风洞试验", "objective": "验证安全放行闭环", "model_code": "SC-001", "planned_condition": "Ma 0.5 / alpha 5 deg", "owner": "试验负责人"})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision := int(release["release"].(map[string]any)["revision"].(float64))
	if _, err := doJSONStatus(client, http.MethodPost, base+"/api/releases/"+id+"/envelope", map[string]any{"request_id": "selfcheck-bad-envelope", "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "speed_min": 20, "speed_max": 500, "attack_angle_min": -5, "attack_angle_max": 5, "load_limit": 80, "temperature_limit": 50}, http.StatusBadRequest); err != nil {
		return shutdownWith(server, done, fmt.Errorf("未阻断超限边界: %w", err))
	}
	trial, err := doJSON(client, http.MethodPost, base+"/api/releases/"+id+"/envelope/trial", map[string]any{"request_id": "selfcheck-trial-001", "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "speed_min": 20, "speed_max": 360, "attack_angle_min": -5, "attack_angle_max": 5, "load_limit": 80, "temperature_limit": 50})
	if err != nil || trial["evaluation_status"] != "blocked" {
		return shutdownWith(server, done, fmt.Errorf("边界试算未返回阻断"))
	}
	envelope := post("/api/releases/"+id+"/envelope", map[string]any{"request_id": "selfcheck-envelope-001", "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "speed_min": 20, "speed_max": 180, "attack_angle_min": -5, "attack_angle_max": 10, "load_limit": 80, "temperature_limit": 50})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision = int(envelope["release"].(map[string]any)["revision"].(float64))
	channel := func(kind string, min, max float64, n int) {
		post("/api/releases/"+id+"/channels", map[string]any{"request_id": "selfcheck-channel-" + kind, "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "id": "ch-" + kind, "channel_type": kind, "sensor_code": "S-" + kind, "range_min": min, "range_max": max, "calibrated_at": time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), "expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339), "evidence_digest": "digest-" + kind})
		if err2 == nil {
			revision++
		}
		_ = n
	}
	channel("pressure", 0, 200, 1)
	channel("strain", -100, 100, 2)
	channel("torque", -50, 50, 3)
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	batchChannels, batchErr := doJSON(client, http.MethodPost, base+"/api/releases/"+id+"/channels/batch", map[string]any{"request_id": "selfcheck-channel-batch", "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "channels": []any{map[string]any{"id": "ch-pressure", "channel_type": "pressure", "sensor_code": "S-pressure", "range_min": 0, "range_max": 200, "calibrated_at": time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), "expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339), "evidence_digest": "digest-pressure"}, map[string]any{"id": "ch-strain", "channel_type": "strain", "sensor_code": "S-strain", "range_min": -100, "range_max": 100, "calibrated_at": time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), "expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339), "evidence_digest": "digest-strain"}, map[string]any{"id": "ch-torque", "channel_type": "torque", "sensor_code": "S-torque", "range_min": -50, "range_max": 50, "calibrated_at": time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), "expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339), "evidence_digest": "digest-torque"}}})
	if batchErr != nil {
		return shutdownWith(server, done, batchErr)
	}
	_ = batchChannels
	revision++
	post("/api/releases/"+id+"/channels/confirm", map[string]any{"request_id": "selfcheck-channels-confirm", "expected_revision": revision, "actor": "测控工程师", "role": "engineer"})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision++
	drill := func(kind string) {
		post("/api/releases/"+id+"/drills", map[string]any{"request_id": "selfcheck-drill-" + kind, "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "id": "dr-" + kind, "interlock_type": kind, "performed_by": "测控工程师", "performed_at": time.Now().UTC().Format(time.RFC3339), "result": "passed", "observed_response_ms": 120, "evidence_digest": "drill-" + kind})
		if err2 == nil {
			revision++
		}
	}
	drill("emergency_stop")
	drill("overlimit_cutoff")
	drill("data_loss")
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	post("/api/releases/"+id+"/drills/confirm", map[string]any{"request_id": "selfcheck-drills-confirm", "expected_revision": revision, "actor": "测控工程师", "role": "engineer", "review_id": "review-selfcheck"})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision++
	post("/api/releases/"+id+"/witness", map[string]any{"request_id": "selfcheck-witness", "expected_revision": revision, "actor": "安全见证员", "role": "witness", "reviewer": "安全见证员", "observations": "证据已抽查", "issues": []any{map[string]any{"id": "issue-1", "description": "核对现场标识"}}})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision++
	post("/api/releases/"+id+"/witness/remediation", map[string]any{"request_id": "selfcheck-remediate", "expected_revision": revision, "actor": "试验负责人", "role": "owner", "issue_id": "issue-1", "evidence": "现场照片与复核记录"})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision++
	post("/api/releases/"+id+"/witness/issue", map[string]any{"request_id": "selfcheck-accept", "expected_revision": revision, "actor": "安全见证员", "role": "witness", "reviewer": "安全见证员", "issue_id": "issue-1", "action": "accept"})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision++
	post("/api/releases/"+id+"/witness/sign", map[string]any{"request_id": "selfcheck-sign", "expected_revision": revision, "actor": "安全见证员", "role": "witness", "reviewer": "安全见证员", "signed_revision": revision})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	revision++
	checklist, err2 := doJSON(client, http.MethodGet, base+"/api/releases/"+id+"/checklist", nil)
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	post("/api/releases/"+id+"/authorize", map[string]any{"request_id": "selfcheck-authorize", "expected_revision": revision, "actor": "授权人", "role": "authorizer", "authorizer": "授权人", "signed_revision": revision, "checklist_digest": checklist["digest"]})
	if err2 != nil {
		return shutdownWith(server, done, err2)
	}
	evidence, err := doRaw(client, http.MethodGet, base+"/api/releases/"+id+"/evidence", nil)
	if err != nil {
		return shutdownWith(server, done, err)
	}
	if len(evidence) < 100 {
		return shutdownWith(server, done, fmt.Errorf("证据包为空"))
	}
	_, err = doJSONStatus(client, http.MethodPost, base+"/api/releases/"+id+"/envelope", map[string]any{"request_id": "selfcheck-after-release", "expected_revision": revision + 1, "actor": "测控工程师", "role": "engineer", "speed_min": 20, "speed_max": 100, "attack_angle_min": -5, "attack_angle_max": 5, "load_limit": 50, "temperature_limit": 40}, http.StatusLocked)
	if err != nil {
		return shutdownWith(server, done, err)
	}
	return shutdownWith(server, done, nil)
}

func waitHealth(client *http.Client, url string) error {
	for i := 0; i < 30; i++ {
		if _, err := doRaw(client, http.MethodGet, url, nil); err == nil {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("健康检查超时")
}
func shutdownWith(server *http.Server, done <-chan error, err error) error {
	shutdownErr := server.Shutdown(context.Background())
	if serveErr := <-done; serveErr != nil && serveErr != http.ErrServerClosed && shutdownErr == nil {
		shutdownErr = serveErr
	}
	if err != nil {
		return err
	}
	return shutdownErr
}
