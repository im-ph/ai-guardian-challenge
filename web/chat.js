// chat.js - 聊天页面逻辑
let conversationId = null;
let isProcessing = false;
let siteInfo = null;
let pendingImageUrl = null;
let captchaVerified = false;

// ========== Thinking 过滤器状态 ==========
// 用于在流式接收 AI 回复时，实时过滤 <think>...</think> 标签内的内容
let isInsideThinkTag = false;      // 当前是否在 <think> 标签内部
let thinkTagBuffer = '';           // 用于检测不完整的标签片段

const urlParams = new URLSearchParams(window.location.search);
const isNewChat = urlParams.get('new') === '1';
const existingId = urlParams.get('id');

async function init() {
    try {
        const response = await fetch('/api/info');
        siteInfo = await response.json();

        if (isNewChat) {
            createNewConversation();
        } else if (existingId) {
            conversationId = existingId;
            await loadConversation();
        } else {
            window.location.href = '/user.html';
        }
    } catch (error) {
        console.error('初始化失败:', error);
        showCustomAlert('初始化失败，请重试');
        setTimeout(() => {
            window.location.href = '/user.html';
        }, 2000);
    }
}

// 简易验证码
function doSimpleCaptcha() {
    const btn = document.getElementById('simpleCaptchaChatBtn');
    btn.textContent = '验证中...';
    btn.disabled = true;
    setTimeout(() => {
        captchaVerified = true;
        btn.textContent = '验证成功 ✓';
        btn.classList.add('verified');
    }, 800);
}

function createNewConversation() {
    const modal = document.getElementById('newChatModal');
    modal.classList.add('active');
}

async function confirmNewChat() {
    if (!captchaVerified) {
        showCustomAlert('请完成人机验证');
        return;
    }

    try {
        const response = await fetch('/api/conversation/new', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ turnstileToken: 'simple-verified' })
        });

        const data = await response.json();

        if (data.success) {
            conversationId = data.conversationId;
            document.getElementById('newChatModal').classList.remove('active');

            const messagesDiv = document.getElementById('chatMessages');
            messagesDiv.innerHTML = '';
            addMessage('assistant', data.initialMessage);

            updateTurnCounter(0, 20);

            document.getElementById('sendBtn').disabled = false;
            document.getElementById('messageInput').disabled = false;
            document.getElementById('messageInput').focus();
        } else {
            showCustomAlert(data.error || '创建对话失败');
        }
    } catch (error) {
        console.error('创建对话失败:', error);
        showCustomAlert('创建对话失败，请重试');
    }
}

function showCustomAlert(message, isSuccess = false) {
    const modal = document.createElement('div');
    modal.className = 'custom-alert-overlay';
    modal.innerHTML = `
        <div class="custom-alert ${isSuccess ? 'success' : ''}">
            <div class="custom-alert-icon">${isSuccess ? '🎉' : '⚠️'}</div>
            <div class="custom-alert-message">${message}</div>
            <button class="custom-alert-btn" onclick="this.closest('.custom-alert-overlay').remove()">确定</button>
        </div>
    `;
    document.body.appendChild(modal);

    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    });
}

async function loadConversation() {
    try {
        const response = await fetch(`/api/conversation/${conversationId}`);

        if (!response.ok) {
            throw new Error('对话不存在');
        }

        const conversation = await response.json();

        const messagesDiv = document.getElementById('chatMessages');
        messagesDiv.innerHTML = '';

        conversation.messages.forEach(msg => {
            addMessage(msg.role, msg.content);
        });

        updateTurnCounter(conversation.turnCount, conversation.maxTurns);

        const sendBtn = document.getElementById('sendBtn');
        const messageInput = document.getElementById('messageInput');

        if (!conversation.isActive) {
            showStatus('此对话已结束', 'warning');
            sendBtn.disabled = true;
            messageInput.disabled = true;
        } else {
            sendBtn.disabled = false;
            messageInput.disabled = false;
            messageInput.focus();
        }

        if (conversation.isSuccess) {
            showStatus(`🎉 恭喜！你已成功获取口令：${conversation.foundPassword}`, 'success');
        }
    } catch (error) {
        console.error('加载对话失败:', error);
        const messagesDiv = document.getElementById('chatMessages');
        messagesDiv.innerHTML = '<div class="message assistant"><div class="message-content">对话不存在或已被删除</div></div>';
        document.getElementById('sendBtn').disabled = true;
        document.getElementById('messageInput').disabled = true;
    }
}

function addMessage(role, content) {
    const messagesDiv = document.getElementById('chatMessages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${role}`;

    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';

    const imgMatch = content.match(/\[图片:(\/Pic\/[^\]]+)\]/);
    if (imgMatch) {
        const imgUrl = imgMatch[1];
        const textOnly = content.replace(/\[图片:\/Pic\/[^\]]+\]\n?/, '').trim();

        const img = document.createElement('img');
        img.src = imgUrl;
        img.className = 'message-image';
        img.alt = '用户上传的图片';
        img.onclick = () => window.open(imgUrl, '_blank');
        contentDiv.appendChild(img);

        if (textOnly) {
            const textNode = document.createTextNode(textOnly);
            contentDiv.appendChild(textNode);
        }
    } else {
        contentDiv.textContent = content;
    }

    messageDiv.appendChild(contentDiv);
    messagesDiv.appendChild(messageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function updateTurnCounter(current, max) {
    document.getElementById('turnCounter').textContent = `剩余轮数: ${max - current}/${max}`;
}

function showStatus(message, type) {
    const statusDiv = document.getElementById('chatStatus');
    statusDiv.textContent = message;
    statusDiv.className = `chat-status ${type}`;
    statusDiv.style.display = 'block';
}

function updateCharCounter() {
    const input = document.getElementById('messageInput');
    const counter = document.getElementById('charCounter');
    if (!input || !counter) return;
    const len = input.value.length;
    counter.textContent = `${len}/3000`;
    counter.classList.remove('warning', 'over');
    if (len > 3000) {
        counter.classList.add('over');
    } else if (len > 2400) {
        counter.classList.add('warning');
    }
}

/**
 * filterThinkingContent - 过滤 AI 输出中的 <think>...</think> 内容
 * 在流式接收过程中逐片段调用，维护 isInsideThinkTag 状态
 * @param {string} chunk - AI 输出的增量文本片段
 * @returns {string} 过滤后的文本（不含 think 标签内容）
 */
function filterThinkingContent(chunk) {
    let result = '';
    let text = thinkTagBuffer + chunk;
    thinkTagBuffer = '';

    let i = 0;
    while (i < text.length) {
        if (isInsideThinkTag) {
            // 在 think 标签内部，寻找 </think>
            const closeIdx = text.indexOf('</think>', i);
            if (closeIdx !== -1) {
                // 找到关闭标签，跳过 think 内容
                isInsideThinkTag = false;
                i = closeIdx + '</think>'.length;
            } else {
                // 未找到关闭标签，可能标签被截断，缓存末尾部分
                // 保留最后 8 个字符（</think> 长度）以防截断
                if (text.length - i > 8) {
                    // 丢弃已确认在 think 内部的内容
                    thinkTagBuffer = text.slice(text.length - 8);
                } else {
                    thinkTagBuffer = text.slice(i);
                }
                break;
            }
        } else {
            // 不在 think 标签内，寻找 <think>
            const openIdx = text.indexOf('<think>', i);
            if (openIdx !== -1) {
                // 找到开始标签，输出标签之前的内容
                result += text.slice(i, openIdx);
                isInsideThinkTag = true;
                i = openIdx + '<think>'.length;
            } else {
                // 检查是否可能有截断的 <think 标签
                // 检查末尾是否以 < 开头且可能是 <think> 的前缀
                let possiblePartial = '';
                for (let j = Math.max(i, text.length - 7); j < text.length; j++) {
                    const remaining = text.slice(j);
                    if ('<think>'.startsWith(remaining)) {
                        possiblePartial = remaining;
                        result += text.slice(i, j);
                        thinkTagBuffer = possiblePartial;
                        i = text.length;
                        break;
                    }
                }
                if (!possiblePartial) {
                    result += text.slice(i);
                    i = text.length;
                }
                break;
            }
        }
    }

    return result;
}

function autoResizeTextarea() {
    const textarea = document.getElementById('messageInput');
    if (!textarea) return;
    textarea.style.height = 'auto';
    const maxHeight = 150;
    const newHeight = Math.min(textarea.scrollHeight, maxHeight);
    textarea.style.height = newHeight + 'px';
    textarea.style.overflowY = textarea.scrollHeight > maxHeight ? 'auto' : 'hidden';
}

function handleImageSelect(e) {
    const file = e.target.files[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
        showCustomAlert('请选择图片文件');
        return;
    }

    if (file.size > 10 * 1024 * 1024) {
        showCustomAlert('图片大小不能超过10MB');
        return;
    }

    const reader = new FileReader();
    reader.onload = (ev) => {
        const preview = document.getElementById('imagePreview');
        const previewImg = document.getElementById('previewImg');
        previewImg.src = ev.target.result;
        preview.style.display = 'block';
    };
    reader.readAsDataURL(file);

    uploadImage(file);
}

async function uploadImage(file) {
    const formData = new FormData();
    formData.append('image', file);
    if (conversationId) {
        formData.append('conversationId', conversationId);
    }

    const preview = document.getElementById('imagePreview');

    try {
        preview.classList.add('uploading');
        const response = await fetch('/api/upload-image', {
            method: 'POST',
            body: formData
        });

        const data = await response.json();

        if (response.ok) {
            pendingImageUrl = data.url;
            preview.classList.remove('uploading');
        } else {
            showCustomAlert(data.error || '图片上传失败');
            removeImage();
        }
    } catch (error) {
        console.error('图片上传失败:', error);
        showCustomAlert('图片上传失败，请重试');
        removeImage();
    }
}

function removeImage() {
    pendingImageUrl = null;
    const preview = document.getElementById('imagePreview');
    preview.style.display = 'none';
    preview.classList.remove('uploading');
    document.getElementById('previewImg').src = '';
    document.getElementById('imageInput').value = '';
}

async function sendMessage() {
    const input = document.getElementById('messageInput');
    const message = input.value.trim();

    if ((!message && !pendingImageUrl) || isProcessing) return;

    if (message.length > 3000) {
        showCustomAlert('输入字符超过3000字');
        return;
    }

    isProcessing = true;
    document.getElementById('sendBtn').disabled = true;
    input.disabled = true;

    const imageUrl = pendingImageUrl;
    const displayContent = imageUrl ? `[图片:${imageUrl}]\n${message}` : message;

    addMessage('user', displayContent);
    input.value = '';
    updateCharCounter();
    input.style.height = 'auto';
    input.style.overflowY = 'hidden';
    removeImage();

    // 重置 Thinking 过滤器状态（每次新消息开始时重置）
    isInsideThinkTag = false;
    thinkTagBuffer = '';

    const messagesDiv = document.getElementById('chatMessages');
    const aiMessageDiv = document.createElement('div');
    aiMessageDiv.className = 'message assistant';
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    aiMessageDiv.appendChild(contentDiv);
    messagesDiv.appendChild(aiMessageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;

    try {
        const response = await fetch('/api/conversation/message', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ conversationId, message, imageUrl })
        });

        if (!response.ok) {
            const error = await response.json();

            if (error.type === 'user_success') {
                contentDiv.remove();
                aiMessageDiv.remove();
                showCustomAlert(error.message, true);
                showStatus(`🎉 ${error.message}`, 'success');
                document.getElementById('sendBtn').disabled = true;
                input.disabled = true;
                isProcessing = false;
                return;
            }

            contentDiv.textContent = error.error || '发送失败';

            if (error.foundPassword) {
                showStatus(`检测到口令：${error.foundPassword}，对话已结束`, 'warning');
                document.getElementById('sendBtn').disabled = true;
                input.disabled = true;
            }

            isProcessing = false;
            return;
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let fullText = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value);
            const lines = chunk.split('\n');

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.substring(6);

                    if (data === '[DONE]') {
                        break;
                    }

                    try {
                        const parsed = JSON.parse(data);

                        if (parsed.type === 'content') {
                            // 过滤 <think>...</think> 标签内的思考内容
                            const filteredContent = filterThinkingContent(parsed.content);
                            if (filteredContent) {
                                fullText += filteredContent;
                                contentDiv.textContent = fullText;
                                messagesDiv.scrollTop = messagesDiv.scrollHeight;
                            }
                        } else if (parsed.type === 'password_found') {
                            document.getElementById('sendBtn').disabled = true;
                            input.disabled = true;

                            // 从已显示的内容中移除口令文本，防止口令在聊天记录中可见
                            if (parsed.password) {
                                fullText = fullText.replace(parsed.password, '***');
                                // 清理福利机制追加的引导语（支持新旧格式）
                                fullText = fullText.replace(/\n\n好吧，你已经和我聊了这么久了[^]*?口令是：\*\*\*/g, '');
                                contentDiv.textContent = fullText;
                            }

                            setTimeout(() => {
                                if (parsed.isFirstWinner) {
                                    showCustomAlert(`🎉🎉🎉 恭喜你成功拿到${parsed.prizeType}口令！\n\n口令是：${parsed.password}\n\n请联系管理员QQ：${siteInfo.adminQQ} 微信：${siteInfo.adminWechat}兑奖（${parsed.prizeAmount}红包）`, true);
                                    showStatus(`🎉 恭喜获得${parsed.prizeType}！口令：${parsed.password}`, 'success');
                                } else {
                                    showCustomAlert(`你成功得到了${parsed.prizeType}口令：${parsed.password}，但是已有用户抢先了，再试试吧！`, false);
                                    showStatus('口令已被使用，继续尝试！', 'warning');
                                }
                            }, 1000);
                        } else if (parsed.type === 'bonus_offer') {
                            // 福利口令二选一弹窗
                            showBonusChoiceModal(parsed);
                        } else if (parsed.type === 'error') {
                            contentDiv.textContent = parsed.content;
                            showStatus('发送失败', 'error');
                        }
                    } catch (e) {
                        // 忽略解析错误
                    }
                }
            }
        }

        // 更新轮次信息
        try {
            const response = await fetch(`/api/conversation/${conversationId}`);
            if (response.ok) {
                const conversation = await response.json();
                updateTurnCounter(conversation.turnCount, conversation.maxTurns);

                if (!conversation.isActive) {
                    document.getElementById('sendBtn').disabled = true;
                    document.getElementById('messageInput').disabled = true;
                    if (conversation.isSuccess) {
                        showStatus(`🎉 恭喜！你已成功获取口令：${conversation.foundPassword}`, 'success');
                    } else {
                        showStatus('对话已结束', 'warning');
                    }
                }
            }
        } catch (e) {
            console.error('更新轮数失败:', e);
        }

    } catch (error) {
        console.error('发送失败:', error);
        contentDiv.textContent = '发送失败，请重试';
        showStatus('发送失败', 'error');
    } finally {
        isProcessing = false;
        const sendBtn = document.getElementById('sendBtn');
        const messageInput = document.getElementById('messageInput');

        fetch(`/api/conversation/${conversationId}`)
            .then(res => res.json())
            .then(conv => {
                if (conv.isActive && conv.turnCount < conv.maxTurns) {
                    sendBtn.disabled = false;
                    messageInput.disabled = false;
                    messageInput.focus();
                }
            })
            .catch(err => {
                console.error('检查对话状态失败:', err);
                sendBtn.disabled = false;
                messageInput.disabled = false;
            });
    }
}

// showBonusChoiceModal 显示福利口令二选一弹窗
// 当用户总对话轮次达到阈值时弹出，让用户选择：领取福利口令 or 放弃继续挑战主口令
function showBonusChoiceModal(bonusData) {
    // 创建遮罩层
    const overlay = document.createElement('div');
    overlay.style.cssText = `
        position: fixed; top: 0; left: 0; width: 100%; height: 100%;
        background: rgba(0,0,0,0.6); z-index: 10000;
        display: flex; align-items: center; justify-content: center;
        backdrop-filter: blur(4px);
    `;

    // 弹窗主体
    const modal = document.createElement('div');
    modal.style.cssText = `
        background: linear-gradient(135deg, rgba(30,40,80,0.95), rgba(20,30,60,0.98));
        border: 1px solid rgba(100,180,255,0.3);
        border-radius: 16px; padding: 32px; max-width: 440px; width: 90%;
        box-shadow: 0 20px 60px rgba(0,0,0,0.5), 0 0 30px rgba(100,180,255,0.15);
        color: #e0e8ff; text-align: center;
        animation: bonusModalIn 0.4s ease-out;
    `;

    // 注入动画样式
    if (!document.getElementById('bonusModalStyle')) {
        const style = document.createElement('style');
        style.id = 'bonusModalStyle';
        style.textContent = `
            @keyframes bonusModalIn {
                from { transform: scale(0.8) translateY(20px); opacity: 0; }
                to { transform: scale(1) translateY(0); opacity: 1; }
            }
            .bonus-btn {
                display: block; width: 100%; padding: 14px 20px; margin: 10px 0;
                border: none; border-radius: 10px; font-size: 16px; font-weight: 600;
                cursor: pointer; transition: all 0.2s ease;
            }
            .bonus-btn:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0,0,0,0.3); }
            .bonus-btn-claim {
                background: linear-gradient(135deg, #00c853, #00e676);
                color: #fff;
            }
            .bonus-btn-continue {
                background: linear-gradient(135deg, #2979ff, #448aff);
                color: #fff;
            }
        `;
        document.head.appendChild(style);
    }

    modal.innerHTML = `
        <div style="font-size: 42px; margin-bottom: 12px;">🎁</div>
        <h2 style="margin: 0 0 8px; font-size: 22px; color: #80b0ff;">恭喜触发福利！</h2>
        <p style="margin: 0 0 16px; font-size: 14px; color: #8899bb;">
            你已累计对话 <strong style="color: #ffcc00;">${bonusData.totalTurns}</strong> 轮，触发了福利口令彩蛋！
        </p>
        <div style="background: rgba(255,255,255,0.06); border-radius: 10px; padding: 14px; margin-bottom: 20px; text-align: left;">
            <p style="margin: 0 0 8px; font-size: 14px; color: #aabbdd;">🎯 你可以选择：</p>
            <p style="margin: 0 0 6px; font-size: 13px; color: #80e0a0;">
                <strong>选项一：</strong>立即领取福利口令（奖品：${bonusData.consolationPrizeAmount}），对话结束。
            </p>
            <p style="margin: 0; font-size: 13px; color: #80b0ff;">
                <strong>选项二：</strong>放弃福利口令，继续挑战主口令（累计到80轮自动获得），奖品更丰厚！
            </p>
        </div>
        <p style="margin: 0 0 16px; font-size: 12px; color: #ff8866;">
            ⚠️ 注意：奖品只能二选一，选择后不可更改！
        </p>
        <button class="bonus-btn bonus-btn-claim" id="bonusClaimBtn">🎉 领取福利口令（${bonusData.consolationPrizeAmount}）</button>
        <button class="bonus-btn bonus-btn-continue" id="bonusContinueBtn">🔥 放弃福利，继续挑战主口令</button>
    `;

    overlay.appendChild(modal);
    document.body.appendChild(overlay);

    // 领取福利口令
    document.getElementById('bonusClaimBtn').addEventListener('click', async () => {
        try {
            const resp = await fetch('/api/conversation/bonus-choice', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ conversationId, choice: 'claim' })
            });
            const result = await resp.json();
            overlay.remove();

            if (result.success) {
                // 禁用输入
                document.getElementById('sendBtn').disabled = true;
                document.getElementById('messageInput').disabled = true;

                // 展示获奖弹窗
                if (result.isFirstWinner) {
                    showCustomAlert(`🎉🎉🎉 恭喜！你领取了福利口令！\n\n口令是：${result.password}\n\n请联系管理员QQ：${siteInfo.adminQQ} 微信：${siteInfo.adminWechat}兑奖（${result.prizeAmount}红包）`, true);
                } else {
                    showCustomAlert(`你领取了福利口令：${result.password}，但已有用户抢先了，再试试吧！`, false);
                }
                showStatus('🎉 已领取福利口令，对话结束', 'success');
            } else {
                showCustomAlert(result.error || '操作失败', false);
            }
        } catch (err) {
            overlay.remove();
            showCustomAlert('网络错误，请重试', false);
        }
    });

    // 放弃福利口令，继续挑战
    document.getElementById('bonusContinueBtn').addEventListener('click', async () => {
        try {
            const resp = await fetch('/api/conversation/bonus-choice', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ conversationId, choice: 'continue' })
            });
            const result = await resp.json();
            overlay.remove();

            if (result.success) {
                showCustomAlert('🔥 你选择了继续挑战主口令！\n\n当总对话轮次达到80次时将自动获得主口令，加油！', false);
                showStatus('继续挑战主口令中...', 'info');
            } else {
                showCustomAlert(result.error || '操作失败', false);
            }
        } catch (err) {
            overlay.remove();
            showCustomAlert('网络错误，请重试', false);
        }
    });
}

document.getElementById('sendBtn').addEventListener('click', sendMessage);

document.getElementById('messageInput').addEventListener('input', () => {
    updateCharCounter();
    autoResizeTextarea();
});

document.getElementById('messageInput').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    }
});

document.getElementById('uploadBtn').addEventListener('click', () => {
    document.getElementById('imageInput').click();
});

document.getElementById('imageInput').addEventListener('change', handleImageSelect);

document.getElementById('removeImage').addEventListener('click', removeImage);

init();
