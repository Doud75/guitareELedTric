// On importe les fonctions Go exposées dans app.go
// Le chemin est relatif à renderer.js
import { GetControllers, GetDetails, Reload } from '../wailsjs/go/main/App';

window.addEventListener('DOMContentLoaded', () => {
    const listEl = document.getElementById('controllers');
    const tbody = document.querySelector('#details tbody');

    async function loadControllers() {
        try {
            listEl.innerHTML = '';
            // On appelle directement la fonction importée
            const ips = await GetControllers();
            if (!ips) return;

            ips.forEach(ip => {
                const li = document.createElement('li');
                li.textContent = ip;
                li.style.cursor = "pointer";
                li.style.padding = "5px";
                li.onclick = async () => {
                    tbody.innerHTML = '<tr><td colspan="2">Chargement...</td></tr>';
                    // On appelle directement la fonction importée
                    const details = await GetDetails(ip);
                    tbody.innerHTML = '';
                    if (!details) return;

                    details.forEach(d => {
                        const tr = document.createElement('tr');
                        tr.innerHTML = `<td>${d.universe}</td><td>${d.ranges.join(', ')}</td>`;
                        tbody.appendChild(tr);
                    });
                };
                listEl.appendChild(li);
            });
        } catch (err) {
            console.error("Erreur au chargement des contrôleurs:", err);
        }
    }

    document.getElementById('reload').onclick = async () => {
        try {
            await Reload();
            tbody.innerHTML = '';
            await loadControllers();
        } catch (err) {
            console.error("Erreur lors du rechargement:", err);
        }
    };

    // Lancement initial
    loadControllers();
});