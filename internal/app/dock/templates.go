package dock

const layoutTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Gin Auth Demo</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
            width: 100%;
            max-width: 450px;
            animation: slideUp 0.5s ease;
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(30px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 30px;
            text-align: center;
        }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header p { opacity: 0.9; font-size: 14px; }
        .form-container { padding: 40px 30px; }
        .form-group { margin-bottom: 20px; }
        .form-group label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 500;
            font-size: 14px;
        }
        .form-group input {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #e0e0e0;
            border-radius: 10px;
            font-size: 15px;
            transition: all 0.3s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }
        .btn {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 10px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
        }
        .btn:active { transform: translateY(0); }
        .links {
            text-align: center;
            margin-top: 20px;
            color: #666;
            font-size: 14px;
        }
        .links a {
            color: #667eea;
            text-decoration: none;
            font-weight: 600;
        }
        .links a:hover { text-decoration: underline; }
        .alert {
            padding: 12px 15px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-size: 14px;
            display: none;
        }
        .alert.error {
            background: #fee;
            color: #c33;
            border: 1px solid #fcc;
            display: block;
        }
        .alert.success {
            background: #efe;
            color: #3c3;
            border: 1px solid #cfc;
            display: block;
        }
        .dashboard {
            max-width: 800px;
            width: 100%;
        }
        .nav {
            background: white;
            padding: 20px 30px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-radius: 15px;
            margin-bottom: 30px;
        }
        .nav-brand {
            font-size: 24px;
            font-weight: bold;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .nav-user {
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .nav-user span { color: #666; }
        .btn-logout {
            padding: 8px 20px;
            background: #ff4757;
            color: white;
            border: none;
            border-radius: 20px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s;
        }
        .btn-logout:hover {
            background: #ee3742;
            transform: scale(1.05);
        }
        .card {
            background: white;
            border-radius: 15px;
            padding: 30px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .card h2 {
            color: #333;
            margin-bottom: 15px;
            font-size: 20px;
        }
        .card p { color: #666; line-height: 1.6; }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }
        .info-item {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
        }
        .info-item h3 {
            color: #667eea;
            font-size: 24px;
            margin-bottom: 5px;
        }
        .info-item p { color: #999; font-size: 14px; }
    </style>
</head>
<body>
    {{template "content" .}}
</body>
</html>`

const loginTemplate = `{{define "content"}}
<div class="container">
    <div class="header">
        <h1>👋 欢迎回来</h1>
        <p>登录您的账户以继续</p>
    </div>
    <div class="form-container">
        <div id="alert" class="alert"></div>
        <form id="loginForm">
            <div class="form-group">
                <label>邮箱地址</label>
                <input type="email" name="email" required placeholder="your@email.com">
            </div>
            <div class="form-group">
                <label>密码</label>
                <input type="password" name="password" required placeholder="••••••••">
            </div>
            <button type="submit" class="btn">登录</button>
        </form>
        <div class="links">
            还没有账户？ <a href="/register">立即注册</a>
        </div>
    </div>
</div>

<script>
document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const data = Object.fromEntries(formData);
    
    try {
        const res = await fetch('/api/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        });
        const result = await res.json();
        
        if (res.ok) {
            showAlert('success', '登录成功！正在跳转...');
            setTimeout(() => window.location.href = '/dashboard', 500);
        } else {
            showAlert('error', result.error || '登录失败');
        }
    } catch (err) {
        showAlert('error', '网络错误，请重试');
    }
});

function showAlert(type, msg) {
    const alert = document.getElementById('alert');
    alert.className = 'alert ' + type;
    alert.textContent = msg;
}
</script>
{{end}}`

const registerTemplate = `{{define "content"}}
<div class="container">
    <div class="header">
        <h1>🚀 创建账户</h1>
        <p>开始您的旅程</p>
    </div>
    <div class="form-container">
        <div id="alert" class="alert"></div>
        <form id="registerForm">
            <div class="form-group">
                <label>用户名</label>
                <input type="text" name="username" required placeholder="johndoe" minlength="3">
            </div>
            <div class="form-group">
                <label>邮箱地址</label>
                <input type="email" name="email" required placeholder="your@email.com">
            </div>
            <div class="form-group">
                <label>密码</label>
                <input type="password" name="password" required placeholder="至少6位字符" minlength="6">
            </div>
            <button type="submit" class="btn">注册</button>
        </form>
        <div class="links">
            已有账户？ <a href="/login">立即登录</a>
        </div>
    </div>
</div>

<script>
document.getElementById('registerForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const data = Object.fromEntries(formData);
    
    try {
        const res = await fetch('/api/register', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        });
        const result = await res.json();
        
        if (res.ok) {
            showAlert('success', '注册成功！正在跳转...');
            setTimeout(() => window.location.href = '/dashboard', 500);
        } else {
            showAlert('error', result.error || '注册失败');
        }
    } catch (err) {
        showAlert('error', '网络错误，请重试');
    }
});

function showAlert(type, msg) {
    const alert = document.getElementById('alert');
    alert.className = 'alert ' + type;
    alert.textContent = msg;
}
</script>
{{end}}`

const dashboardTemplate = `{{define "content"}}
<div class="dashboard">
    <div class="nav">
        <div class="nav-brand">🔐 Gin Auth</div>
        <div class="nav-user">
            <span>👤 {{.Username}}</span>
            <button class="btn-logout" onclick="logout()">退出登录</button>
        </div>
    </div>
    
    <div class="card">
        <h2>欢迎回来，{{.Username}}！</h2>
        <p>您已成功登录系统。这是一个基于 Gin Framework 的 Session 认证示例应用。</p>
        
        <div class="info-grid">
            <div class="info-item">
                <h3>🔒</h3>
                <p>安全认证</p>
            </div>
            <div class="info-item">
                <h3>⚡</h3>
                <p>高性能</p>
            </div>
            <div class="info-item">
                <h3>🚀</h3>
                <p>现代化</p>
            </div>
        </div>
    </div>
    
    <div class="card">
        <h2>会话信息</h2>
        <p><strong>用户ID:</strong> {{.UserID}}</p>
        <p style="margin-top: 10px;"><strong>登录时间:</strong> {{.LoginTime}}</p>
    </div>
</div>

<script>
async function logout() {
    if (!confirm('确定要退出登录吗？')) return;
    
    try {
        const res = await fetch('/api/logout', {method: 'POST'});
        if (res.ok) {
            window.location.href = '/login';
        }
    } catch (err) {
        alert('退出失败，请重试');
    }
}
</script>
{{end}}`
