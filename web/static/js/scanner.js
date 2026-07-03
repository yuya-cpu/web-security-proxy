document.addEventListener("DOMContentLoaded", () => {
    const runBtn = document.getElementById("active-scan-run");
    const statusEl = document.getElementById("active-scan-status");
    const resultsEl = document.getElementById("active-scan-results");

    if (!runBtn || !statusEl || !resultsEl) {
        return;
    }

    runBtn.addEventListener("click", async () => {
        const transactionID = runBtn.dataset.transactionId;
        if (!transactionID) {
            return;
        }

        statusEl.textContent = "スキャン中...";
        statusEl.className = "status-msg";
        runBtn.disabled = true;
        resultsEl.className = "scan-results";
        resultsEl.textContent = "スキャンを実行しています...";

        try {
            const response = await fetch(`/api/transactions/${transactionID}/active-scan`, {
                method: "POST",
            });
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || "スキャンに失敗しました");
            }

            resultsEl.innerHTML = renderScanReport(data);
            statusEl.textContent = `完了 (Overall Risk: ${data.overall_risk})`;
        } catch (error) {
            statusEl.textContent = error.message;
            statusEl.className = "status-msg error";
            resultsEl.className = "scan-results empty";
            resultsEl.textContent = error.message;
        } finally {
            runBtn.disabled = false;
        }
    });
});

function renderScanReport(report) {
    const findings = (report.findings || [])
        .map((finding) => {
            const desc = finding.description ? `<span>${escapeHTML(finding.description)}</span>` : "";
            return `<li class="finding"><span class="badge risk-${finding.risk_level}">${finding.risk_level}</span> <strong>${escapeHTML(finding.title)}</strong> ${desc}</li>`;
        })
        .join("");

    const resources = (report.resources || [])
        .map((resource) => {
            const content = resource.found && resource.content
                ? `<pre>${escapeHTML(resource.content)}</pre>`
                : `<p class="empty">見つかりませんでした (status: ${resource.status_code || "n/a"})</p>`;
            return `<div class="detail-block"><h3>${escapeHTML(resource.name)}</h3><p class="mono">${escapeHTML(resource.url)}</p>${content}</div>`;
        })
        .join("");

    return `
        <div class="detail-block">
            <h3>Overall Risk</h3>
            <p><span class="badge risk-${report.overall_risk}">${report.overall_risk}</span></p>
        </div>
        <div class="detail-block">
            <h3>Findings</h3>
            <ul class="finding-list">${findings || '<li class="empty">結果なし</li>'}</ul>
        </div>
        ${resources}
    `;
}

function escapeHTML(value) {
    return String(value)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll("\"", "&quot;");
}
