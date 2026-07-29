// =========================================================================
// 1. УПРАВЛЕНИЕ МОДАЛЬНЫМ ОКНОМ ДЕТАЛЕЙ ОБОРУДОВАНИЯ (TAILWIND UI)
// =========================================================================
window.openHardwareModal = function(guid, fqdn) {
    document.getElementById("modalTitle").innerText = "📋 Конфигурация железа: " + fqdn;
    const modal = document.getElementById("hwModal");
    const body = document.getElementById("modalBody");
    body.innerHTML = '<div class="text-slate-500 italic text-center py-4">Загрузка спецификаций из SQLite...</div>';
    modal.classList.remove("hidden");

    fetch("/details?guid=" + guid)
        .then(response => response.json())
        .then(data => {
            // Вспомогательная функция для генерации HTML-кода одной секции аккордеона
            const createAccordion = (id, title, icon, contentHTML) => {
                return `
                    <div class="border border-slate-200 rounded-lg overflow-hidden bg-slate-50/50">
                        <button onclick="toggleAccordion('${id}')"
                                class="w-full flex items-center justify-between px-4 py-3 bg-slate-100/80 hover:bg-slate-100 text-slate-700 font-semibold text-sm transition-colors focus:outline-none">
                            <div class="flex items-center space-x-2">
                                <span>${icon}</span>
                                <span>${title}</span>
                            </div>
                            <svg id="icon-${id}" class="h-4 w-4 transform transition-transform duration-200 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                            </svg>
                        </button>
                        <div id="content-${id}" class="hidden p-4 bg-white text-slate-600 border-t border-slate-100 text-xs sm:text-sm leading-relaxed">
                            ${contentHTML}
                        </div>
                    </div>
                `;
            };

            // Формируем контент для каждого блока
            let html = "";

            html += createAccordion("os", "Операционная система", "🪟", data.os || 'N/A');
            html += createAccordion("ram", "Оперативная память", "📊", data.ram || 'N/A');
            html += createAccordion("motherboard", "Материнская плата", "🧱", data.motherboard || 'N/A');
          html += createAccordion("bios", "Версия BIOS", "📟", data.bios || 'N/A');

          const l1KB = data.cpu_l1 ? (data.cpu_l1 / 1024).toFixed(0) : "0";
                      const l2KB = data.cpu_l2 ? (data.cpu_l2 / 1024).toFixed(0) : "0";
                      const l3MB = data.cpu_l3 ? (data.cpu_l3 / (1024 * 1024)).toFixed(0) : "0";
                      const speedGHz = data.cpu_speed ? (data.cpu_speed / 1000).toFixed(2) : "0";

                      const cpuTableHTML = `
                          <div class="space-y-3">
                              <div class="text-sm font-bold text-slate-800 border-b border-slate-100 pb-2">${data.cpu_name || 'N/A'}</div>
                              <div class="grid grid-cols-2 gap-2 text-xs sm:text-sm">
                                  <div class="flex py-1.5 border-b border-slate-100"><span class="w-40 font-semibold text-slate-500">Физические ядра:</span><span class="font-medium text-slate-800">${data.cpu_cores || '0'}</span></div>
                                  <div class="flex py-1.5 border-b border-slate-100"><span class="w-44 font-semibold text-slate-500">Логические процессоры:</span><span class="font-medium text-slate-800">${data.cpu_threads || '0'}</span></div>
                                  <div class="flex py-1.5 border-b border-slate-100"><span class="w-40 font-semibold text-slate-500">Макс. частота:</span><span class="font-medium text-slate-800 font-mono">${data.cpu_speed || '0'} MHz (~${speedGHz} GHz)</span></div>
                                  <div class="flex py-1.5 border-b border-slate-100"><span class="w-44 font-semibold text-slate-500">Кэш память L1:</span><span class="font-medium text-slate-800 font-mono">${l1KB} KB</span></div>
                                  <div class="flex py-1.5 border-b border-slate-100"><span class="w-40 font-semibold text-slate-500">Кэш память L2:</span><span class="font-medium text-slate-800 font-mono">${l2KB} KB</span></div>
                                  <div class="flex py-1.5 border-b border-slate-100"><span class="w-44 font-semibold text-slate-500">Кэш память L3:</span><span class="font-medium text-slate-800 font-mono">${l3MB} MB</span></div>
                              </div>
                          </div>
                      `;
                      html += createAccordion("cpu", "Процессор (CPU)", "🧠", cpuTableHTML);

            if (data.gpus && data.gpus.length > 0) {
                html += createAccordion("gpus", "Видеокарты", "📺", data.gpus.join('<br>'));
            } else {
                html += createAccordion("gpus", "Видеокарты", "📺", 'Встроенное графическое ядро или данные отсутствуют');
            }

            // =========================================================================
            // ИСПРАВЛЕНО: СЕКЦИЯ НАКОПИТЕЛЕЙ С ИНДИКАТОРАМИ ЗАПОЛНЕННОСТИ (PROGRESS BARS)
            // =========================================================================
            if (data.disks && data.disks.length > 0) {
                let disksHTML = '<div class="space-y-4">';

                data.disks.forEach(disk => {
                    const totalGB = disk.total_bytes / (1024 * 1024 * 1024);
                    const freeGB = disk.free_bytes / (1024 * 1024 * 1024);
                    const usedGB = totalGB - freeGB;

                    // Защита от деления на ноль
                    const usedPercent = totalGB > 0 ? Math.round((usedGB / totalGB) * 100) : 0;
                    const freePercent = 100 - usedPercent;

                    // Если свободного места меньше 10%, красим шкалу в предупреждающий красный цвет
                    const barColor = freePercent < 10 ? 'bg-red-500' : 'bg-blue-600';

                    disksHTML += `
                        <div class="bg-slate-50 p-3 rounded-lg border border-slate-200">
                            <!-- Верхняя строчка: Буква и модель -->
                            <div class="flex justify-between items-center mb-1">
                                <span class="font-bold text-slate-800">${disk.drive} [${disk.type}] <span class="font-normal text-slate-500 text-xs">${disk.vendor}</span></span>
                                <span class="text-xs font-semibold text-slate-600">${usedPercent}% Занято</span>
                            </div>

                            <!-- Графическая шкала заполненности (Progress Bar) -->
                            <div class="w-full bg-slate-200 rounded-full h-2.5 mb-2 overflow-hidden shadow-inner">
                                <div class="${barColor} h-2.5 rounded-full transition-all duration-500" style="width: ${usedPercent}%"></div>
                            </div>

                            <!-- Нижняя строчка: Подробная емкость в гигабайтах -->
                            <div class="flex justify-between text-[11px] font-mono text-slate-500">
                                <span>Занято: ${usedGB.toFixed(1)} GB (${usedPercent}%)</span>
                                <span>Свободно: ${freeGB.toFixed(1)} GB (${freePercent}%)</span>
                                <span class="font-bold text-slate-700">Всего: ${totalGB.toFixed(1)} GB</span>
                            </div>
                        </div>
                    `;
                });

                disksHTML += '</div>';
                html += createAccordion("disks", "Накопители", "💾", disksHTML);
            } else {
                html += createAccordion("disks", "Накопители", "💾", '<div class="text-slate-400 italic">Диски не найдены</div>');
            }

            // =========================================================================
            // ДОБАВЛЕНО: АККОРДЕОН СЕТЕВЫХ ИНТЕРФЕЙСОВ
            // =========================================================================
            if (data.networks && data.networks.length > 0) {
                            let netsRowsHTML = "";

                            data.networks.forEach(net => {
                                // Формируем красивый цветной значок статуса
                                const isUp = (net.status || "").toLowerCase() === "up";
                                const statusDot = isUp
                                    ? '<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800"><span class="h-1.5 w-1.5 rounded-full bg-green-500 mr-1.5"></span>Up</span>'
                                    : '<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800"><span class="h-1.5 w-1.5 rounded-full bg-red-500 mr-1.5"></span>Down</span>';

                                // Подсветка типа получения адреса (DHCP / Static)
                                const isDhcp = (net.ip_type || "").toLowerCase().includes("dhcp");
                                const typeBadge = isDhcp
                                    ? '<span class="px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded border border-blue-100 font-mono text-[10px] uppercase font-bold">DHCP</span>'
                                    : '<span class="px-1.5 py-0.5 bg-amber-50 text-amber-600 rounded border border-amber-100 font-mono text-[10px] uppercase font-bold">Static</span>';

                                // Разбиваем IP-адреса через запятую на отдельные строки для читаемости
                                const ipList = net.ips ? net.ips.split(',').join('<br>') : '—';

                                    netsRowsHTML += `
                                        <tr class="hover:bg-slate-50 transition-colors">
                                            <td class="px-3 py-2 font-medium text-slate-900 border-b border-slate-100">${net.name}</td>
                                            <!-- ИСПРАВЛЕНО: Читаем net.mac, который теперь строго сопоставлен с JSON тегом бэкенда! -->
                                            <td class="px-3 py-2 font-mono text-slate-500 border-b border-slate-100 text-xs">${net.mac || '—'}</td>
                                            <td class="px-3 py-2 font-mono text-slate-600 border-b border-slate-100 text-xs leading-relaxed">${ipList}</td>
                                            <td class="px-3 py-2 border-b border-slate-100">${statusDot}</td>
                                            <td class="px-3 py-2 border-b border-slate-100">${typeBadge}</td>
                                        </tr>
                                    `;
                            });

                            const netsTableHTML = `
                                <div class="overflow-x-auto border border-slate-200 rounded-lg shadow-sm">
                                    <table class="w-full text-left text-xs border-collapse">
                                        <thead>
                                            <tr class="bg-slate-700 text-white uppercase text-[10px] tracking-wider">
                                                <th class="px-3 py-2 font-semibold">Имя интерфейса</th>
                                                <th class="px-3 py-2 font-semibold">МАК-адрес</th>
                                                <th class="px-3 py-2 font-semibold">IP-адресы</th>
                                                <th class="px-3 py-2 font-semibold">Статус</th>
                                                <th class="px-3 py-2 font-semibold">Тип (IPType)</th>
                                            </tr>
                                        </thead>
                                        <tbody class="divide-y divide-slate-100 bg-white">
                                            ${netsRowsHTML}
                                        </tbody>
                                    </table>
                                </div>
                            `;

                            html += createAccordion("networks", "Сетевые интерфейсы", "🌐", netsTableHTML);
                        } else {
                            html += createAccordion("networks", "Сетевые интерфейсы", "🌐", '<div class="text-slate-400 italic">Сетевые адаптеры не найдены</div>');
                        }

            body.innerHTML = `<div class="space-y-3">${html}</div>`;

            // Автоматически открываем первый блок (Операционная система) для лучшего UX
            toggleAccordion("os");
        })
        .catch(err => {
            body.innerHTML = '<span class="text-red-500 font-medium block text-center py-4">Ошибка загрузки данных: ' + err + '</span>';
        });
}

// Глобальная функция для переключения состояния аккордеона (Развернуть / Свернуть)
window.toggleAccordion = function(id) {
	const content = document.getElementById("content-" + id);
	const icon = document.getElementById("icon-" + id);

	if (content.classList.contains("hidden")) {
		content.classList.remove("hidden");
		icon.classList.add("rotate-180");
	} else {
		content.classList.add("hidden");
		icon.classList.remove("rotate-180");
	}
}

window.closeModal = function() { document.getElementById("hwModal").classList.add("hidden"); }
window.onclick = function(event) {
    const modal = document.getElementById("hwModal");
    if (event.target == modal) modal.classList.add("hidden");
}

const getStatusIndicator = (statusText) => {
    const text = (statusText || "").toLowerCase();
    const isRunning = text.includes("running");
    const isNotFound = text.includes("not found") || text.includes("not installed") || text === "n/a" || text === "";

    if (isRunning) {
        return '<div class="flex items-center space-x-2"><span class="relative flex h-2.5 w-2.5"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span><span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500"></span></span><span class="text-xs font-semibold text-green-700">Running</span></div>';
    } else if (isNotFound) {
        return '<div class="flex items-center space-x-2"><span class="inline-flex rounded-full h-2.5 w-2.5 bg-slate-400"></span><span class="text-xs font-normal text-slate-400">Not Installed</span></div>';
    } else {
        return '<div class="flex items-center space-x-2"><span class="inline-flex rounded-full h-2.5 w-2.5 bg-red-500"></span><span class="text-xs font-normal text-slate-500">Stopped</span></div>';
    }
};

// =========================================================================
// 2. ЖАДНЫЙ ИНТЕРАКТИВНЫЙ ПОИСК (МГНОВЕННАЯ КЛИЕНТСКАЯ ФИЛЬТРАЦИЯ)
// =========================================================================
const searchInput = document.getElementById("searchQuery");
const clearBtn = document.getElementById("clearSearchBtn");

// Функция "жадного" перебора строк прямо в DOM-дереве
function performGreedySearch() {
    const query = searchInput.value.trim().toLowerCase();
    const rows = document.querySelectorAll("#tableBody tr:not(#noDataRow)");
    let hasVisibleRows = false;

    // Показываем/скрываем кнопку сброса
    if (query !== "") {
        clearBtn.classList.remove("hidden");
    } else {
        clearBtn.classList.add("hidden");
    }

    rows.forEach(row => {
        // Извлекаем весь текст из ячеек текущей строки (ФИО, Логин, FQDN, IP)
        const rowText = row.innerText.toLowerCase();

        // Если строка содержит поисковый запрос — убираем 'hidden', иначе — скрываем
        if (rowText.includes(query)) {
            row.classList.remove("hidden");
            hasVisibleRows = true;
        } else {
            row.classList.add("hidden");
        }
    });

    // Управляем отображением строки "Ничего не найдено"
    let noDataRow = document.getElementById("noDataRow");
    if (!hasVisibleRows) {
        if (!noDataRow) {
            noDataRow = document.createElement("tr");
            noDataRow.id = "noDataRow";
            noDataRow.innerHTML = '<td colspan="7" class="text-center text-slate-400 py-10 italic">Записи не найдены по вашему запросу</td>';
            document.getElementById("tableBody").appendChild(noDataRow);
        }
    } else if (noDataRow) {
        noDataRow.remove();
    }
}

// Вешаем живой слушатель на ввод букв (срабатывает мгновенно, без задержек)
searchInput.addEventListener("input", performGreedySearch);

window.resetLiveSearch = function() {
    searchInput.value = "";
    clearBtn.classList.add("hidden");
    performGreedySearch();
}

// =========================================================================
// 3. СЛУШАТЕЛЬ REAL-TIME ОБНОВЛЕНИЙ SSE
// =========================================================================
const eventSource = new EventSource("/events");

eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    const tbody = document.getElementById("tableBody");
    const noDataRow = document.getElementById("noDataRow");

    if (noDataRow) noDataRow.remove();

    const existingRow = document.getElementById("row-" + data.guid);
    const rdpHTML = getStatusIndicator(data.rdp);
    const vncHTML = getStatusIndicator(data.vnc);

    let targetRow;

    if (existingRow) {
        existingRow.querySelector(".cell-name").innerText = data.name;
        existingRow.querySelector(".cell-login code").innerText = data.login;
        existingRow.querySelector(".fqdn-link").innerText = data.fqdn;
        existingRow.querySelector(".cell-ip").innerText = data.ip;
        existingRow.querySelector(".cell-rdp").innerHTML = rdpHTML;
        existingRow.querySelector(".cell-vnc").innerHTML = vncHTML;
        existingRow.querySelector(".cell-time strong").innerText = data.time;

        targetRow = existingRow;
    } else {
        const tr = document.createElement("tr");
        tr.id = "row-" + data.guid;
        tr.innerHTML = `
            <td class="cell-name px-6 py-4 font-medium text-slate-900">${data.name}</td>
            <td class="cell-login px-6 py-4"><code class="px-2 py-1 bg-slate-100 rounded text-xs text-red-600 font-mono">${data.login}</code></td>
            <td class="cell-fqdn px-6 py-4"><span class="fqdn-link text-blue-600 hover:text-blue-800 font-semibold cursor-pointer border-b border-dashed border-blue-500 hover:border-solid transition-all" onclick="openHardwareModal('${data.guid}', '${data.fqdn}')">${data.fqdn}</span></td>
            <td class="cell-ip px-6 py-4 text-slate-600 font-mono">${data.ip}</td>
            <td class="cell-rdp px-6 py-4">${rdpHTML}</td>
            <td class="cell-vnc px-6 py-4">${vncHTML}</td>
            <td class="cell-time px-6 py-4 text-slate-500 font-medium"><strong>${data.time}</strong></td>
        `;
        tbody.insertBefore(tr, tbody.firstChild);
        targetRow = tr;
    }

    // Запускаем красивую вспышку активности
    targetRow.classList.remove("hover:bg-slate-50/80");
    targetRow.className = "bg-green-100 hover:bg-slate-50 transition-all duration-1000";

    // ИСПРАВЛЕНО: Сразу же прогоняем новую строку через "жадный" фильтр!
    // Если она не подходит под текущий ввод админа — она бесшумно скроется в фоне, не мешая обзору.
    const query = searchInput.value.trim().toLowerCase();
    if (query !== "" && !targetRow.innerText.toLowerCase().includes(query)) {
        targetRow.classList.add("hidden");
    }

    setTimeout(() => {
        if (!targetRow.classList.contains("hidden")) {
            targetRow.className = "hover:bg-slate-50/80 transition-colors duration-500";
        }
    }, 1500);

    if (existingRow) {
        tbody.insertBefore(existingRow, tbody.firstChild);
    }
};

eventSource.onerror = function() {
    console.log("SSE соединение потеряно. Ожидание восстановления...");
};
