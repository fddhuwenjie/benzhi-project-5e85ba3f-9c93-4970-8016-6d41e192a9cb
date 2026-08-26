# BENZHI_README

## 项目说明
- 项目：benzhi-project-5e85ba3f-9c93-4970-8016-6d41e192a9cb
- 项目用途：风洞试验安全放行台提供单机浏览器工作台，将风洞试验从建档、运行边界评估、测量链核验、联锁演练、独立见证整改推进到授权放行和不可变证据封存；服务仅监听可配置的 127.0.0.1 高位端口，并通过真实 HTTP selfcheck 验证完整闭环。
- Go 工具链：`golang:1.23.0`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 项目描述
- 项目名称：风洞试验安全放行台
- 项目介绍：面向空气动力试验团队的单机浏览器工作台，将一次风洞试验从任务建档、边界评估、测量链核验、联锁演练、独立见证推进到授权放行与证据封存，并以明确状态机阻止缺少安全证据的试验进入可执行状态。项目根目录提供简体中文 README.md，说明用途以及标准构建、运行和测试方式。
- 项目概述：面向空气动力试验团队的单机浏览器工作台，将一次风洞试验从任务建档、边界评估、测量链核验、联锁演练、独立见证推进到授权放行与证据封存，并以明确状态机阻止缺少安全证据的试验进入可执行状态。项目根目录提供简体中文 README.md，说明用途以及标准构建、运行和测试方式。
- 核心工作流：试验负责人创建草案并登记模型与工况边界，测控工程师完成风险条件、测量通道和联锁演练核验，安全见证员审查问题及整改证据，全部门禁通过后签署放行并封存不可变证据包。
- 对外接口：由 Go 服务提供原生 HTML、CSS 和 JavaScript 的浏览器工作台，包含试验概览、分步核验表单、证据附件元数据、问题整改、见证签署、放行摘要和审计时间线，并通过同源 JSON 接口完成全部状态操作。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-5e85ba3f-9c93-4970-8016-6d41e192a9cb-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-5e85ba3f-9c93-4970-8016-6d41e192a9cb-arm64 linux/arm64

docker run -it benzhi-project-5e85ba3f-9c93-4970-8016-6d41e192a9cb-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
