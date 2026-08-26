const $ = (q) => document.querySelector(q);
const state = { list: [], summary: null };
const stages = [
  ["draft", "边界评估", "测控工程师"],
  ["measurement_verification", "测量链核验", "测控工程师"],
  ["interlock_drill", "联锁演练", "测控工程师"],
  ["witness_review", "独立见证", "安全见证员 / 负责人"],
  ["pending_authorization", "授权签署", "授权人"],
  ["released", "证据封存", "只读"]
];
const labels = {pressure:"压力",strain:"应变",torque:"力矩",emergency_stop:"急停",overlimit_cutoff:"超限切断",data_loss:"数据失联"};
const esc = (v) => String(v ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
const requestID = () => crypto.randomUUID();
const meta = (role, actor) => ({request_id:requestID(), expected_revision:state.summary.release.revision, role, actor});
const isoNow = () => new Date().toISOString();
const isoDate = (days) => new Date(Date.now() + days*86400000).toISOString();

async function api(path, options={}) {
  const response = await fetch(path, {headers:{"Content-Type":"application/json", ...(options.headers||{})}, ...options});
  const type = response.headers.get("content-type") || "";
  const body = type.includes("json") ? await response.json() : await response.text();
  if (!response.ok) {
    const error = new Error(body?.error?.message || `请求失败 (${response.status})`);
    error.code = body?.error?.code;
    throw error;
  }
  return body;
}

function notify(message, success=false) {
  const node = $("#notice"); node.textContent=message; node.className=`notice${success?" success":""}`;
  window.setTimeout(()=>node.classList.add("hidden"), 5000);
}

async function refreshList(selectID) {
  const page = await api("/api/releases?limit=100"); state.list=page.items;
  const list=$("#releaseList"); list.replaceChildren();
  if (!page.items.length) { const p=document.createElement("p");p.className="muted";p.textContent="尚无试验档案";list.append(p);return; }
  page.items.forEach(r=>{const button=document.createElement("button");button.className=`release-item${state.summary?.release.id===r.id?" active":""}`;button.innerHTML=`<strong>${esc(r.title)}</strong><span>${esc(r.model_code)} · ${esc(statusName(r.status))} · r${r.revision}</span>`;button.onclick=()=>selectRelease(r.id);list.append(button);});
  if (selectID) await selectRelease(selectID);
}

async function selectRelease(id) {
  try { state.summary=await api(`/api/releases/${encodeURIComponent(id)}`); render(); await loadAudit(); await refreshList(); }
  catch (err) { notify(errorMessage(err)); }
}

function statusName(status) { return stages.find(s=>s[0]===status)?.[1] || status; }
function errorMessage(err) { return err.code==="revision_conflict" ? "档案已被其他操作更新，请刷新后重试。" : err.message; }

function render() {
  const s=state.summary, r=s.release; $("#emptyState").classList.add("hidden");$("#releaseView").classList.remove("hidden");
  $("#releaseCode").textContent=`${r.model_code} · ${r.id}`;$("#releaseTitle").textContent=r.title;$("#releaseMeta").textContent=`责任人 ${r.owner} · ${r.planned_condition}`;
  $("#statusBadge").textContent=s.status_label;$("#statusBadge").className=`status ${r.status}`;$("#revisionBadge").textContent=`revision ${r.revision}`;
  const index=stages.findIndex(x=>x[0]===r.status);$("#steps").innerHTML=stages.map((x,i)=>`<div class="step ${i<index?"done":i===index?"current":""}">${i+1}. ${x[1]}</div>`).join("");
  $("#stageTitle").textContent=stages[index][1];$("#roleTag").textContent=stages[index][2];
  const gates=$("#gateList");gates.replaceChildren();(s.pending_gates.length?s.pending_gates:["全部门禁已通过"]).forEach((g,i)=>{const li=document.createElement("li");li.textContent=g;if(!s.pending_gates.length)li.className="clear";gates.append(li)});if(["interlock_drill","witness_review","pending_authorization"].includes(r.status)){const li=document.createElement("li"),b=document.createElement("button");b.className="secondary";b.textContent="安全回退到测量核验";b.onclick=rollback;li.append(b);gates.append(li)}
  $("#profileSummary").innerHTML=`<dt>模型</dt><dd>${esc(r.model_code)}</dd><dt>目标</dt><dd>${esc(r.objective)}</dd><dt>工况</dt><dd>${esc(r.planned_condition)}</dd><dt>创建</dt><dd>${new Date(r.created_at).toLocaleString()}</dd>`;
  renderStage(r);
}

function renderStage(r) {
  const body=$("#stageBody");
  if(r.status==="draft") body.innerHTML=draftTemplate(r);
  if(r.status==="measurement_verification") body.innerHTML=channelTemplate(r);
  if(r.status==="interlock_drill") body.innerHTML=drillTemplate(r);
  if(r.status==="witness_review") body.innerHTML=witnessTemplate(r);
  if(r.status==="pending_authorization") body.innerHTML=authorizationTemplate(r);
  if(r.status==="released") body.innerHTML=releasedTemplate(r);
  bindStage(r);
}

function draftTemplate(r) {
  const e=r.envelope||{speed_min:20,speed_max:180,attack_angle_min:-10,attack_angle_max:15,load_limit:80,temperature_limit:60,violations:[]};
  const violations=e.violations?.length?`<div class="callout">${e.violations.map(v=>`<div>${esc(v)}</div>`).join("")}</div>`:"";
  return `<div class="callout">合格限制：速度不超过 350 m/s，攻角绝对值不超过 25 deg，载荷不超过 120 kN，温度为 10–85 C。</div>${violations}<form id="envelopeForm"><div class="fields"><label>最低速度 (m/s)<input name="speed_min" type="number" step="0.1" value="${e.speed_min}" required></label><label>最高速度 (m/s)<input name="speed_max" type="number" step="0.1" value="${e.speed_max}" required></label><label>最小攻角 (deg)<input name="attack_angle_min" type="number" step="0.1" value="${e.attack_angle_min}" required></label><label>最大攻角 (deg)<input name="attack_angle_max" type="number" step="0.1" value="${e.attack_angle_max}" required></label><label>载荷限制 (kN)<input name="load_limit" type="number" step="0.1" value="${e.load_limit}" required></label><label>温度限制 (C)<input name="temperature_limit" type="number" step="0.1" value="${e.temperature_limit}" required></label></div><div id="trialResult"></div><div class="actions"><button type="button" id="trialEnvelope" class="secondary">试算方案</button><button type="button" id="editProfile" class="secondary">维护档案</button><button>确认并提交边界</button></div></form>`;
}

function channelTemplate(r) {
  const rows=(r.channels||[]).map(c=>`<tr><td>${labels[c.channel_type]}</td><td>${esc(c.sensor_code)}</td><td>${c.range_min} – ${c.range_max}</td><td>${c.verification_status==="passed"?"合格":"阻断"}</td></tr>`).join("")||`<tr><td colspan="4">尚未登记通道</td></tr>`;
  return `<table class="data-table"><thead><tr><th>类型</th><th>传感器</th><th>量程</th><th>核验</th></tr></thead><tbody>${rows}</tbody></table><form id="channelForm"><div class="fields"><label>通道类型<select name="channel_type"><option value="pressure">压力</option><option value="strain">应变</option><option value="torque">力矩</option></select></label><label>传感器编号<input name="sensor_code" required></label><label>量程下限<input name="range_min" type="number" step="0.1" value="-120" required></label><label>量程上限<input name="range_max" type="number" step="0.1" value="400" required></label><label>校准时间<input name="calibrated_at" type="datetime-local" required></label><label>有效期<input name="expires_at" type="datetime-local" required></label></div><label>校准证据摘要<input name="evidence_digest" required placeholder="校准证书编号或 SHA-256"></label><div class="actions"><button>登记 / 更新通道</button><button type="button" id="batchChannels" class="secondary">批量核验已有通道</button><button type="button" id="confirmChannels">确认测量链</button></div></form>`;
}

function drillTemplate(r) {
  const rows=(r.drills||[]).map(d=>`<tr><td>${labels[d.interlock_type]}</td><td>${d.result==="passed"?"通过":"失败"}</td><td>${d.observed_response_ms} ms</td><td>${esc(d.performed_by)}</td></tr>`).join("")||`<tr><td colspan="4">尚未记录演练</td></tr>`;
  return `<table class="data-table"><thead><tr><th>联锁</th><th>结果</th><th>响应</th><th>执行人</th></tr></thead><tbody>${rows}</tbody></table><form id="drillForm"><div class="fields"><label>联锁类型<select name="interlock_type"><option value="emergency_stop">急停</option><option value="overlimit_cutoff">超限切断</option><option value="data_loss">数据失联</option></select></label><label>执行人<input name="performed_by" required></label><label>结果<select name="result"><option value="passed">通过</option><option value="failed">失败</option></select></label><label>观测响应 (ms)<input name="observed_response_ms" type="number" min="1" max="10000" value="120" required></label></div><label>演练证据摘要<input name="evidence_digest" required></label><div class="actions"><button>记录演练</button><button type="button" id="confirmDrills">提交独立见证</button></div></form>`;
}

function witnessTemplate(r) {
  const w=r.witness;
  if(!w.reviewer) return `<form id="witnessReviewForm"><label>见证员<input name="reviewer" required></label><label>审查观察<textarea name="observations" required></textarea></label><label>问题清单（每行一项，可留空）<textarea name="issues" placeholder="例如：急停按钮标识需要复核"></textarea></label><div class="actions"><button>提交审查结论</button></div></form>`;
  const issues=w.issues||[]; const rows=issues.map(i=>`<tr><td>${esc(i.id)}</td><td>${esc(i.description)}</td><td>${i.status==="pending_verification"?"待核验":i.closed?"已关闭":"待整改"}</td><td>${esc(i.remediation_evidence||"—")}</td><td>${i.status==="pending_verification"?`<button type="button" class="resolveIssue" data-id="${esc(i.id)}">接受</button>`:i.closed?`<button type="button" class="reopenIssue secondary" data-id="${esc(i.id)}">重开</button>`:""}</td></tr>`).join("")||`<tr><td colspan="5">无见证问题</td></tr>`;
  const open=issues.filter(i=>!i.closed), actionable=open.filter(i=>i.status!=="pending_verification");
  return `<div class="callout ${open.length?"":"good"}">见证员 ${esc(w.reviewer)}：${open.length?`仍有 ${open.length} 项问题待整改或核验`:"问题已全部关闭，可以签署"}</div><table class="data-table"><thead><tr><th>ID</th><th>问题</th><th>状态</th><th>证据</th><th>操作</th></tr></thead><tbody>${rows}</tbody></table>${actionable.length?`<form id="remediationForm"><label>待整改问题<select name="issue_id">${actionable.map(i=>`<option value="${esc(i.id)}">${esc(i.description)}</option>`).join("")}</select></label><label>整改证据<input name="evidence" required></label><div class="actions"><button>提交整改证据</button></div></form>`:open.length?"<div class=\"callout\">整改证据已提交，等待见证员核验。</div>":`<form id="witnessSignForm"><label>见证签署人<input name="reviewer" value="${esc(w.reviewer)}" required></label><div class="callout">签署将固定当前 revision ${r.revision}，提交后进入待授权状态。</div><div class="actions"><button>签署审查通过</button></div></form>`}`;
}

function authorizationTemplate(r) {
  return `<div class="callout good">全部工程门禁和独立见证已通过。最终签署以当前固定 revision ${r.revision} 为准。</div><div id="checklistView" class="callout">正在加载固定清单…</div><form id="authorizeForm"><label>授权人<input name="authorizer" required></label><label>授权结论<input value="批准执行并封存证据" disabled></label><div class="actions"><button>签署最终放行</button></div></form>`;
}

function releasedTemplate(r) {
  return `<div class="callout good"><strong>已完成安全放行</strong><div>证据包 SHA-256：${esc(r.evidence?.digest)}</div></div><p>档案已封存，业务字段和所有核验记录均为只读。证据包含固定修订、规范审计事件和签署信息。</p><div class="actions"><a href="/api/releases/${encodeURIComponent(r.id)}/evidence" download><button>下载 JSON 证据包</button></a></div>`;
}

function bindStage(r) {
  if($("#envelopeForm")) $("#envelopeForm").onsubmit=submitEnvelope;
  if($("#trialEnvelope")) $("#trialEnvelope").onclick=trialEnvelope;
  if($("#editProfile")) $("#editProfile").onclick=editProfile;
  if($("#channelForm")){const f=$("#channelForm");f.calibrated_at.value=isoDate(-1).slice(0,16);f.expires_at.value=isoDate(365).slice(0,16);f.onsubmit=submitChannel;$("#confirmChannels").onclick=confirmChannels;$("#batchChannels").onclick=batchChannels;}
  if($("#drillForm")){ $("#drillForm").onsubmit=submitDrill;$("#confirmDrills").onclick=confirmDrills; }
  if($("#witnessReviewForm")) $("#witnessReviewForm").onsubmit=submitWitness;
  if($("#remediationForm")) $("#remediationForm").onsubmit=submitRemediation;
  if($("#witnessSignForm")) $("#witnessSignForm").onsubmit=signWitness;
  if($("#authorizeForm")) $("#authorizeForm").onsubmit=authorize;
  if($("#checklistView")) loadChecklist();
  document.querySelectorAll(".resolveIssue").forEach(b=>b.onclick=()=>resolveIssue(b.dataset.id,"accept"));
  document.querySelectorAll(".reopenIssue").forEach(b=>b.onclick=()=>{const reason=prompt("重开理由");if(reason)resolveIssue(b.dataset.id,"reopen",reason)});
}

async function command(path,payload,message) {
  try { const result=await api(path,{method:"POST",body:JSON.stringify(payload)});state.summary={release:result.release,status_label:statusName(result.release.status),pending_gates:[]};await selectRelease(result.release.id);notify(message,true); }
  catch(err){notify(errorMessage(err));if(err.code==="revision_conflict")await selectRelease(state.summary.release.id);}
}
async function rollback(){if(!confirm("确认作废后续快照并回到测量核验？"))return;await command(`/api/releases/${state.summary.release.id}/safety-rollback`,{...meta("engineer","测控工程师"),reason:"校准时效复核失败"},"档案已安全回退");}

async function submitEnvelope(event){event.preventDefault();const f=new FormData(event.target);await command(`/api/releases/${state.summary.release.id}/envelope`,{...meta("engineer","测控工程师"),speed_min:+f.get("speed_min"),speed_max:+f.get("speed_max"),attack_angle_min:+f.get("attack_angle_min"),attack_angle_max:+f.get("attack_angle_max"),load_limit:+f.get("load_limit"),temperature_limit:+f.get("temperature_limit")},"边界评估已记录");}
async function trialEnvelope(){const f=new FormData($("#envelopeForm"));try{const v=await api(`/api/releases/${state.summary.release.id}/envelope/trial`,{method:"POST",body:JSON.stringify({...meta("engineer","测控工程师"),speed_min:+f.get("speed_min"),speed_max:+f.get("speed_max"),attack_angle_min:+f.get("attack_angle_min"),attack_angle_max:+f.get("attack_angle_max"),load_limit:+f.get("load_limit"),temperature_limit:+f.get("temperature_limit")})});$("#trialResult").innerHTML=`<div class="callout ${v.evaluation_status==="passed"?"good":""}">${v.evaluation_status==="passed"?"试算通过，可确认提交":"试算阻断："}${(v.violations||[]).map(esc).join("；")}</div>`}catch(e){notify(errorMessage(e))}}
async function submitChannel(event){event.preventDefault();const f=new FormData(event.target),type=f.get("channel_type");await command(`/api/releases/${state.summary.release.id}/channels`,{...meta("engineer","测控工程师"),id:`channel-${type}`,channel_type:type,sensor_code:f.get("sensor_code"),range_min:+f.get("range_min"),range_max:+f.get("range_max"),calibrated_at:new Date(f.get("calibrated_at")).toISOString(),expires_at:new Date(f.get("expires_at")).toISOString(),evidence_digest:f.get("evidence_digest")},"测量通道已核验");}
async function batchChannels(){const channels=(state.summary.release.channels||[]).map(c=>({id:c.id,channel_type:c.channel_type,sensor_code:c.sensor_code,range_min:c.range_min,range_max:c.range_max,calibrated_at:c.calibrated_at,expires_at:c.expires_at,evidence_digest:c.evidence_digest}));if(!channels.length){notify("请先登记通道");return;}await command(`/api/releases/${state.summary.release.id}/channels/batch`,{...meta("engineer","测控工程师"),channels},"批量通道核验完成");}
async function confirmChannels(){await command(`/api/releases/${state.summary.release.id}/channels/confirm`,meta("engineer","测控工程师"),"测量链已确认");}
async function submitDrill(event){event.preventDefault();const f=new FormData(event.target),type=f.get("interlock_type");await command(`/api/releases/${state.summary.release.id}/drills`,{...meta("engineer","测控工程师"),id:`drill-${type}`,interlock_type:type,performed_by:f.get("performed_by"),performed_at:isoNow(),result:f.get("result"),observed_response_ms:+f.get("observed_response_ms"),evidence_digest:f.get("evidence_digest")},"联锁演练已记录");}
async function confirmDrills(){await command(`/api/releases/${state.summary.release.id}/drills/confirm`,{...meta("engineer","测控工程师"),review_id:`review-${crypto.randomUUID()}`},"待见证快照已生成");}
async function submitWitness(event){event.preventDefault();const f=new FormData(event.target),issues=String(f.get("issues")).split("\n").map(v=>v.trim()).filter(Boolean).map((description,i)=>({id:`issue-${i+1}`,description}));await command(`/api/releases/${state.summary.release.id}/witness`,{...meta("witness",f.get("reviewer")),reviewer:f.get("reviewer"),observations:f.get("observations"),issues},"见证审查已提交");}
async function submitRemediation(event){event.preventDefault();const f=new FormData(event.target);await command(`/api/releases/${state.summary.release.id}/witness/remediation`,{...meta("owner",state.summary.release.owner),issue_id:f.get("issue_id"),evidence:f.get("evidence")},"整改证据已提交，等待见证员核验");}
async function resolveIssue(issueID,action,reason="问题证据已核验"){await command(`/api/releases/${state.summary.release.id}/witness/issue`,{...meta("witness","安全见证员"),reviewer:"安全见证员",issue_id:issueID,action,reason},action==="accept"?"问题已由见证员核销":"问题已重开");}
async function signWitness(event){event.preventDefault();const f=new FormData(event.target),revision=state.summary.release.revision;await command(`/api/releases/${state.summary.release.id}/witness/sign`,{...meta("witness",f.get("reviewer")),reviewer:f.get("reviewer"),signed_revision:revision},"独立见证已签署");}
async function authorize(event){event.preventDefault();const f=new FormData(event.target),revision=state.summary.release.revision;const digest=$("#checklistView")?.dataset.digest||"";await command(`/api/releases/${state.summary.release.id}/authorize`,{...meta("authorizer",f.get("authorizer")),authorizer:f.get("authorizer"),signed_revision:revision,checklist_digest:digest},"最终放行已签署，证据包已封存");}
async function loadChecklist(){try{const c=await api(`/api/releases/${state.summary.release.id}/checklist`);const n=$("#checklistView");n.dataset.digest=c.digest;n.innerHTML=`固定清单摘要：<code>${esc(c.digest.slice(0,16))}</code><br>${c.items.map(i=>esc(i.kind)).join(" · ")}`}catch(e){notify(errorMessage(e))}}

async function editProfile(){
  const r=state.summary.release,title=prompt("试验标题",r.title);if(title===null)return;const objective=prompt("试验目标",r.objective);if(objective===null)return;const condition=prompt("计划工况",r.planned_condition);if(condition===null)return;
  try{const check=await api("/api/releases/precheck",{method:"POST",body:JSON.stringify({release_id:r.id,title,objective,model_code:r.model_code,planned_condition:condition,owner:r.owner})});if(!check.can_proceed){notify("字段预检未通过");return;}const result=await api(`/api/releases/${r.id}/profile`,{method:"PUT",body:JSON.stringify({...meta("owner",r.owner),title,objective,model_code:r.model_code,planned_condition:condition,owner:r.owner,confirm_diff:true})});await selectRelease(result.release.id);notify("档案已更新",true);}catch(err){notify(errorMessage(err));}
}

async function loadAudit(){if(!state.summary)return;const type=$("#auditFilter").value,role=$("#auditRole")?.value||"",actor=$("#auditActor")?.value||"",rev=$("#auditFromRevision")?.value||"",page=await api(`/api/releases/${state.summary.release.id}/audit?limit=200&type=${encodeURIComponent(type)}&role=${encodeURIComponent(role)}&actor=${encodeURIComponent(actor)}&revision_from=${encodeURIComponent(rev)}`),node=$("#auditTimeline");node.replaceChildren();if($("#auditStats"))$("#auditStats").textContent=`匹配 ${page.stats?.total||0} 条 · 失败/退回 ${page.stats?.failure_or_return_count||0} 次 · 操作者 ${page.stats?.distinct_actors||0} 人`;if(!page.items.length){node.textContent="当前筛选没有审计记录";return;}page.items.forEach(e=>{const div=document.createElement("div");div.className="event";const time=document.createElement("time");time.textContent=new Date(e.occurred_at).toLocaleString();const text=document.createElement("div");text.innerHTML=`<strong>${esc(e.type)}</strong>${esc(e.actor)} · revision ${e.revision}<div class="transition">${esc(e.from_status||"开始")} → ${esc(e.to_status)}</div>`;div.append(time,text);node.append(div);});}

$("#newRelease").onclick=()=>$("#createDialog").showModal();
$("#createForm").onsubmit=async(event)=>{event.preventDefault();const submit=event.submitter;if(submit?.value==="cancel"){$("#createDialog").close();return;}const f=new FormData(event.target),owner=f.get("owner"),base={title:f.get("title"),objective:f.get("objective"),model_code:f.get("model_code"),planned_condition:f.get("planned_condition"),owner};try{const check=await api("/api/releases/precheck",{method:"POST",body:JSON.stringify(base)});if(!check.can_proceed){notify("请修正字段后再提交");return;}const duplicate=check.duplicates?.length>0;if(duplicate&&!confirm("存在同模型同工况在途档案，仍要创建吗？"))return;const result=await api("/api/releases",{method:"POST",body:JSON.stringify({...base,request_id:requestID(),expected_revision:0,actor:owner,role:"owner",confirm_duplicate:duplicate})});$("#createDialog").close();event.target.reset();await refreshList(result.release.id);notify("试验档案已创建",true);}catch(err){notify(errorMessage(err));}};
$("#auditFilter").onchange=()=>loadAudit().catch(e=>notify(e.message));
$("#auditRole")?.addEventListener("change",()=>loadAudit().catch(e=>notify(e.message)));$("#auditActor")?.addEventListener("change",()=>loadAudit().catch(e=>notify(e.message)));$("#auditFromRevision")?.addEventListener("change",()=>loadAudit().catch(e=>notify(e.message)));

Promise.all([api("/healthz"),refreshList()]).then(()=>{$("#health").textContent="本地服务正常";$("#health").classList.add("ok")}).catch(err=>{$("#health").textContent="服务异常";notify(err.message)});
