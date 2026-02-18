// app.js - 首页逻辑
let siteInfo = null;
let isLoggedIn = false;
let captchaVerified = false;

// 加载站点信息
async function loadInfo() {
    try {
        const response = await fetch('/api/info');
        siteInfo = await response.json();
        updateCountdown();
        setInterval(updateCountdown, 1000);
        // 动态渲染管理员联系方式
        renderFooterContact();
    } catch (error) {
        console.error('加载站点信息失败:', error);
    }
}

// 检查认证状态
async function checkAuth() {
    try {
        const response = await fetch('/api/check-auth');
        const data = await response.json();
        isLoggedIn = data.isLoggedIn;

        if (isLoggedIn) {
            document.getElementById('startBtn').textContent = '🎮 进入游戏';
        }
    } catch (error) {
        console.error('检查认证失败:', error);
    }
}

// 更新倒计时
function updateCountdown() {
    if (!siteInfo) return;

    const deadline = new Date(siteInfo.deadline).getTime();
    const now = Date.now();
    const diff = deadline - now;

    const countdownEl = document.getElementById('countdown');

    if (diff <= 0) {
        countdownEl.textContent = '🎉 活动已结束';
        document.getElementById('startBtn').disabled = true;
        document.getElementById('startBtn').textContent = '活动已结束';
        return;
    }

    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((diff % (1000 * 60)) / 1000);

    countdownEl.textContent = `${days}天 ${hours}小时 ${minutes}分 ${seconds}秒`;
}

// 根据后端配置动态渲染管理员联系方式
function renderFooterContact() {
    const el = document.getElementById('footerContact');
    if (!el || !siteInfo) return;
    const parts = [];
    if (siteInfo.adminQQ) parts.push(`管理员QQ：${siteInfo.adminQQ}`);
    if (siteInfo.adminEmail) parts.push(`邮箱：<a href="mailto:${siteInfo.adminEmail}">${siteInfo.adminEmail}</a>`);
    if (siteInfo.adminWechat) parts.push(`微信：${siteInfo.adminWechat}`);
    el.innerHTML = parts.join(' | ');
}

// 简易验证码验证
function verifyCaptcha() {
    const btn = document.getElementById('simpleCaptchaBtn');
    btn.textContent = '验证中...';
    btn.disabled = true;

    // 简易验证：模拟一个短暂的延迟
    setTimeout(() => {
        captchaVerified = true;
        btn.textContent = '验证成功 ✓';
        btn.classList.add('verified');
    }, 800);
}

// 开始挑战按钮点击
document.getElementById('startBtn').addEventListener('click', () => {
    if (isLoggedIn) {
        window.location.href = '/user.html';
    } else {
        document.getElementById('loginModal').classList.add('active');
    }
});

// 关闭登录弹窗
function closeLoginModal() {
    document.getElementById('loginModal').classList.remove('active');
}

// 提交登录
async function submitLogin() {
    const contact = document.getElementById('contactInput').value.trim();
    const nickname = document.getElementById('nicknameInput').value.trim();

    if (!contact || !nickname) {
        alert('请填写联系方式和昵称');
        return;
    }

    if (!captchaVerified) {
        alert('请先完成人机验证');
        return;
    }

    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ contact, nickname, captchaToken: 'simple-verified' })
        });

        const data = await response.json();

        if (data.success) {
            window.location.href = '/user.html';
        } else {
            alert(data.error || '登录失败');
        }
    } catch (error) {
        console.error('登录失败:', error);
        alert('网络错误，请重试');
    }
}

// 加载获奖者列表
async function loadWinners(page = 1) {
    try {
        const response = await fetch(`/api/winners?page=${page}&pageSize=5`);
        const result = await response.json();
        const winners = result.data || [];
        const container = document.getElementById('winnersDisplay');

        if (winners.length === 0) {
            container.innerHTML = '<div class="no-winners">暂无获奖者，成为第一个挑战成功的人吧！</div>';
            return;
        }

        container.innerHTML = '';
        winners.forEach(winner => {
            const categoryClass = winner.category || 'consolation-prize';
            const card = document.createElement('div');
            card.className = `winner-card ${categoryClass}`;
            card.onclick = () => {
                window.open(`/conversation.html?id=${winner.conversationId}`, '_blank');
            };

            const badgeText = winner.prizeType === 'grand' ? '🏆 特等奖' : '🎁 安慰奖';

            card.innerHTML = `
                <span class="winner-badge">${badgeText}</span>
                <div class="winner-info">
                    <div class="winner-name">${winner.nickname}</div>
                    <div class="winner-time">${new Date(winner.timestamp).toLocaleString('zh-CN')}</div>
                </div>
            `;
            container.appendChild(card);
        });

        // 分页
        if (result.totalPages > 1) {
            renderPagination('winnersPagination', page, result.totalPages, loadWinners);
        }
    } catch (error) {
        console.error('加载获奖者失败:', error);
    }
}

// 加载公开对话
async function loadPublicConversations(page = 1) {
    try {
        const response = await fetch(`/api/public/conversations?page=${page}&pageSize=15`);
        const result = await response.json();
        const conversations = result.data || [];
        const container = document.getElementById('conversationsList');

        if (conversations.length === 0 && page === 1) {
            container.innerHTML = '<div class="no-data">暂无公开对话记录</div>';
            return;
        }

        container.innerHTML = '';
        conversations.forEach(conv => {
            const card = document.createElement('div');
            card.className = `conversation-card ${conv.isSuccess ? 'success' : ''}`;
            card.onclick = () => {
                window.open(`/conversation.html?id=${conv.id}`, '_blank');
            };

            card.innerHTML = `
                <div class="conversation-header">
                    <span class="conversation-user">${conv.nickname}${conv.isSuccess ? '<span class="success-badge">成功</span>' : ''}</span>
                    <span class="conversation-time">${new Date(conv.createdAt).toLocaleString('zh-CN')}</span>
                </div>
                <div class="conversation-preview">${conv.preview || '对话进行中...'}</div>
            `;
            container.appendChild(card);
        });

        if (result.totalPages > 1) {
            renderPagination('conversationsPagination', page, result.totalPages, loadPublicConversations);
        }
    } catch (error) {
        console.error('加载对话失败:', error);
    }
}

// 通用分页渲染
function renderPagination(containerId, currentPage, totalPages, loadFn) {
    const container = document.getElementById(containerId);
    if (!container) return;

    let html = '';
    html += `<button class="pagination-btn" ${currentPage <= 1 ? 'disabled' : ''} onclick="void(0)">‹</button>`;

    const maxVisible = 5;
    let startPage = Math.max(1, currentPage - Math.floor(maxVisible / 2));
    let endPage = Math.min(totalPages, startPage + maxVisible - 1);
    if (endPage - startPage < maxVisible - 1) {
        startPage = Math.max(1, endPage - maxVisible + 1);
    }

    if (startPage > 1) {
        html += `<button class="pagination-btn" onclick="void(0)">1</button>`;
        if (startPage > 2) html += '<span class="pagination-dots">…</span>';
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="pagination-btn ${i === currentPage ? 'active' : ''}" onclick="void(0)">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += '<span class="pagination-dots">…</span>';
        html += `<button class="pagination-btn" onclick="void(0)">${totalPages}</button>`;
    }

    html += `<button class="pagination-btn" ${currentPage >= totalPages ? 'disabled' : ''} onclick="void(0)">›</button>`;

    container.innerHTML = html;

    // 绑定事件
    container.querySelectorAll('.pagination-btn').forEach(btn => {
        if (btn.disabled) return;
        btn.addEventListener('click', () => {
            const text = btn.textContent.trim();
            if (text === '‹') loadFn(currentPage - 1);
            else if (text === '›') loadFn(currentPage + 1);
            else loadFn(parseInt(text));
        });
    });
}

// 初始化
async function init() {
    await checkAuth();
    await loadInfo();
    loadWinners();
    loadPublicConversations();
}

init();
