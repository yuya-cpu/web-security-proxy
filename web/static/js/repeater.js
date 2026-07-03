document.addEventListener("DOMContentLoaded", () => {
    document.querySelectorAll(".tab").forEach((tab) => {
        tab.addEventListener("click", () => {
            const target = tab.dataset.tab;
            document.querySelectorAll(".tab").forEach((t) => t.classList.remove("active"));
            document.querySelectorAll(".tab-content").forEach((c) => c.classList.remove("active"));
            tab.classList.add("active");
            document.getElementById(`tab-${target}`)?.classList.add("active");
        });
    });

    const form = document.getElementById("repeater-form");
    if (!form) {
        return;
    }

    form.addEventListener("submit", async (event) => {
        event.preventDefault();

        const statusEl = document.getElementById("repeater-status");
        const sendBtn = document.getElementById("repeater-send");
        statusEl.textContent = "送信中...";
        statusEl.className = "status-msg";
        sendBtn.disabled = true;

        const payload = {
            method: document.getElementById("repeater-method").value,
            url: document.getElementById("repeater-url").value,
            headers: document.getElementById("repeater-headers").value,
            body: document.getElementById("repeater-body").value,
        };

        try {
            const response = await fetch("/api/repeater/send", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            });

            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || "送信に失敗しました");
            }

            window.location.href = `/transactions/${data.id}`;
        } catch (error) {
            statusEl.textContent = error.message;
            statusEl.className = "status-msg error";
            sendBtn.disabled = false;
        }
    });
});
