# 🚀 Getting Started with Keycloak SSO

## What is Keycloak?

Keycloak is like having a **professional bouncer** for your applications:
- ✅ One login works for all your services (SSO)
- ✅ Can add "Login with Google/Facebook" buttons  
- ✅ Built-in two-factor authentication (2FA)
- ✅ Professional user management interface
- ✅ Industry-standard security (used by big companies!)

## 📖 Two Ways to Get Started

### Option 1: Quick Start Script (Easiest!)

Just run this command and follow the prompts:

```bash
./keycloak-quickstart.sh
```

The script will:
1. Check if Keycloak is running
2. Guide you through Keycloak setup
3. Install necessary libraries
4. Rebuild your services
5. Test the integration

### Option 2: Manual Setup (Detailed Learning)

Follow the comprehensive guide:

```bash
# Open the detailed guide
open KEYCLOAK-SETUP-GUIDE.md
# or
cat KEYCLOAK-SETUP-GUIDE.md
```

This guide includes:
- 📸 Screenshots and visual explanations
- 📝 Step-by-step instructions
- 🔍 Troubleshooting tips
- 💡 Advanced features you can enable later

## 🎯 What You'll Build

**Before** (What you have now):
```
User → AuthService (custom login) → ProfileService/BillingService
```

**After** (With Keycloak):
```
User → Keycloak (professional login) → AuthService → ProfileService/BillingService
              ↓
         Can also add:
         - Google login
         - Facebook login  
         - 2FA
         - Password reset
         - Much more!
```

## 🏃 Quick Test (After Setup)

### Test 1: Direct Keycloak Login
```
1. Open: http://localhost:8090 (Keycloak Admin)
2. Login: admin / admin
3. Verify realm and client are created
```

### Test 2: Login Through Keycloak
```
1. Open: http://localhost:8080/auth/keycloak/login
2. Login with your Keycloak user
3. Should redirect back to AuthService ✅
```

### Test 3: Full SSO Flow
```
1. Open: http://localhost:8081/dashboard
2. Change URL to: http://localhost:8080/auth/keycloak/login?redirect_uri=http://localhost:8081/auth/callback
3. Login with Keycloak
4. Should see ProfileService dashboard ✅
```

## 📚 Documentation Files

| File | Purpose |
|------|---------|
| `KEYCLOAK-SETUP-GUIDE.md` | Complete setup guide with screenshots |
| `keycloak-quickstart.sh` | Automated setup script |
| `auth-service/keycloak.go` | Keycloak integration code |
| `.env` | Configuration (add your client secret here) |

## ⚙️ Configuration Checklist

After running the setup, make sure you have:

- [  ] Keycloak running on http://localhost:8090
- [  ] Created realm: `micro-dashboard-sso`
- [  ] Created client: `auth-service`
- [  ] Copied client secret to `.env`
- [  ] Set `OIDC_ENABLED=true` in `.env`
- [  ] Created test user in Keycloak
- [  ] Rebuilt AuthService
- [  ] Tested login flow

## 🆘 Need Help?

### Common Issues:

**"Connection refused to Keycloak"**
```bash
# Start Keycloak
docker-compose up -d keycloak

# Wait 30 seconds, then try again
```

**"Client not found"**
- Make sure you created the client with exact ID: `auth-service`
- Check you're in the right realm: `micro-dashboard-sso`

**"Invalid redirect URI"**
- Add `http://localhost:8080/*` to valid redirect URIs
- Add `http://localhost:8081/*` and `http://localhost:8082/*` too

### Check Logs:
```bash
# Keycloak logs
docker logs sso_keycloak -f

# AuthService logs
docker logs auth_service -f
```

## 🎓 Learning Resources

- **Keycloak Docs**: https://www.keycloak.org/documentation
- **OpenID Connect**: https://openid.net/connect/
- **Our Full Guide**: `KEYCLOAK-SETUP-GUIDE.md`

## 🎉 What's Next?

Once Keycloak is working, you can:

1. **Add Social Login** (Google, Facebook, GitHub)
2. **Enable 2FA** (Google Authenticator, SMS)
3. **Customize Theme** (Make it match your brand)
4. **Add More Services** (Register them as clients)
5. **User Roles** (Admin, User, etc.)

All of this is explained in `KEYCLOAK-SETUP-GUIDE.md`!

---

**Ready to start?** Run:
```bash
./keycloak-quickstart.sh
```

or read the full guide:
```bash
open KEYCLOAK-SETUP-GUIDE.md
```

Good luck! 🚀
