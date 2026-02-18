// user.js - 用户对话列表页面逻辑
let currentPage = 1;
const pageSize = 15;

async function loadConversations(page = 1) {
    currentPage = page;
    try {
        const response = await fetch(`/api/conversations?page=${page}&pageSize=${pageSize}`);
        if (response.status === 401) {
            window.location.href = '/';
            return;
        }

        const result = await response.json();
        const conversations = result.data || result;
        const container = document.getElementById('userConversations');
        container.innerHTML = '';

        if (conversations.length === 0 && page === 1) {
            container.innerHTML = '<div class="no-data">暂无对话记录，点击上方按钮开始新对话</div>';
            renderUserPagination(0, 0, 0);
            return;
        }

        conversations.forEach(conv => {
            const card = document.createElement('div');
            card.className = `user-conversation-card ${conv.isSuccess ? 'success' : ''} ${!conv.isActive ? 'inactive' : ''}`;
            card.onclick = () => {
                window.location.href = `/chat.html?id=${conv.id}`;
            };

            const statusText = conv.isSuccess ? '✓ 成功获取口令' :
                !conv.isActive ? '已结束' :
                    `进行中 (${conv.turnCount}/${conv.maxTurns})`;

            card.innerHTML = `
                <div class="conv-card-top">
                    <span class="conv-status ${conv.isSuccess ? 'success' : conv.isActive ? 'active' : 'inactive'}">${statusText}</span>
                    <span class="conv-time">${new Date(conv.createdAt).toLocaleString('zh-CN')}</span>
                </div>
                <div class="conv-preview">${conv.lastMessage || conv.preview || ''}</div>
                ${conv.isSuccess ? `<div class="conv-password">🎉 口令: ${conv.foundPassword}</div>` : ''}
            `;

            container.appendChild(card);
        });

        if (result.totalPages !== undefined) {
            renderUserPagination(result.page, result.totalPages, result.total);
        }
    } catch (error) {
        console.error('加载对话失败:', error);
    }
}

function renderUserPagination(page, totalPages, total) {
    let paginationDiv = document.getElementById('userPagination');
    if (!paginationDiv) {
        paginationDiv = document.createElement('div');
        paginationDiv.id = 'userPagination';
        paginationDiv.className = 'admin-pagination';
        const container = document.getElementById('userConversations');
        container.parentNode.insertBefore(paginationDiv, container.nextSibling);
    }

    if (totalPages <= 1) {
        paginationDiv.innerHTML = total > 0 ? `<span class="page-info">共 ${total} 条记录</span>` : '';
        return;
    }

    let html = '<div class="pagination-controls">';

    html += `<button class="page-btn ${page <= 1 ? 'disabled' : ''}" ${page <= 1 ? 'disabled' : ''} onclick="loadConversations(${page - 1})">
        ‹ 上一页
    </button>`;

    const maxVisible = 5;
    let startPage = Math.max(1, page - Math.floor(maxVisible / 2));
    let endPage = Math.min(totalPages, startPage + maxVisible - 1);
    if (endPage - startPage < maxVisible - 1) {
        startPage = Math.max(1, endPage - maxVisible + 1);
    }

    if (startPage > 1) {
        html += `<button class="page-btn" onclick="loadConversations(1)">1</button>`;
        if (startPage > 2) html += `<span class="page-ellipsis">…</span>`;
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="page-btn ${i === page ? 'active' : ''}" onclick="loadConversations(${i})">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += `<span class="page-ellipsis">…</span>`;
        html += `<button class="page-btn" onclick="loadConversations(${totalPages})">${totalPages}</button>`;
    }

    html += `<button class="page-btn ${page >= totalPages ? 'disabled' : ''}" ${page >= totalPages ? 'disabled' : ''} onclick="loadConversations(${page + 1})">
        下一页 ›
    </button>`;

    html += `</div>`;
    html += `<span class="page-info">第 ${page}/${totalPages} 页 · 共 ${total} 条</span>`;

    paginationDiv.innerHTML = html;
}

document.getElementById('newChatBtn').addEventListener('click', () => {
    window.location.href = '/chat.html?new=1';
});

function logout() {
    if (confirm('确定要退出登录吗？')) {
        fetch('/api/logout', { method: 'POST' })
            .then(() => {
                window.location.href = '/';
            });
    }
}

loadConversations();
