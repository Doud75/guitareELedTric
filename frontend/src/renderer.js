window.addEventListener('DOMContentLoaded', async () => {
    const listEl = document.getElementById('controllers');
    const tbody = document.querySelector('#details tbody');

    async function loadControllers() {
        listEl.innerHTML = '';
        const ips = await window.backend.GetControllers();
        ips.forEach(ip => {
            const li = document.createElement('li');
            li.textContent = ip;
            li.onclick = async () => {
                tbody.innerHTML = '';
                const details = await window.backend.GetDetails(ip);
                details.forEach(d => {
                    const tr = document.createElement('tr');
                    tr.innerHTML = `<td>${d.universe}</td><td>${d.ranges.join(', ')}</td>`;
                    tbody.appendChild(tr);
                });
            };
            listEl.appendChild(li);
        });
    }

    document.getElementById('reload').onclick = async () => {
        await window.backend.Reload();
        tbody.innerHTML = '';
        await loadControllers();
    };

    await loadControllers();
});
