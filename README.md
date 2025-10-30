# Micro-Dashboard Hub 🔐

A complete **Single Sign-On (SSO)** implementation using Go (Gin framework) demonstrating a microservice architecture with one Identity Provider (IdP) and two Service Providers (SPs).

## 🎯 Project Overview

This project demonstrates a production-ready SSO flow where:
- **AuthService** acts as the central Identity Provider (IdP)
- **ProfileService** and **BillingService** act as Service Providers (SPs)
- Users authenticate once and gain access to all services seamlessly

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     User's Browser                          │
└──────────────┬──────────────────────────────────────────────┘
               │
               ├─────────────┐
               │             │
               ▼             ▼
    ┌──────────────┐  ┌──────────────┐
    │ ProfileService│  │BillingService│
    │   (SP1)      │  │   (SP2)      │
    │  Port 8081   │  │  Port 8082   │
    └──────┬───────┘  └──────┬───────┘
           │                 │
           │   Redirects     │
           │   for Auth      │
           └────────┬────────┘
                    │
                    ▼
            ┌──────────────┐
            │ AuthService  │
            │    (IdP)     │
            │  Port 8080   │
            └──────┬───────┘
                   │
                   ▼
            ┌──────────────┐
            │  PostgreSQL  │
            │  Port 5432   │
            └──────────────┘
```

## 🚀 Features

### AuthService (IdP) - Port 8080
- ✅ User registration and login
- ✅ Password hashing with bcrypt
- ✅ Session management with cookies
- ✅ JWT token generation for SPs
- ✅ SSO endpoint that checks existing sessions
- ✅ PostgreSQL database for user storage

### ProfileService (SP1) - Port 8081
- ✅ Protected dashboard route
- ✅ Authentication middleware
- ✅ JWT validation
- ✅ Independent session management
- ✅ Automatic redirect to IdP when unauthenticated

### BillingService (SP2) - Port 8082
- ✅ Protected invoices route
- ✅ Authentication middleware
- ✅ JWT validation
- ✅ Independent session management
- ✅ Automatic redirect to IdP when unauthenticated

## 📋 Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Web browser

## 🔧 Installation & Setup

### 1. Clone the Repository

```bash
cd /Users/wv-supakorn/Documents/my\ project/Micro-Dashboard-Hub
```

### 2. Start All Services

```bash
docker-compose up --build
```

This will start:
- PostgreSQL database on port 5432
- AuthService on port 8080
- ProfileService on port 8081
- BillingService on port 8082

### 3. Wait for Services to Start

The first time you run this, it will:
- Download Docker images
- Build the Go applications
- Initialize the PostgreSQL database
- Start all services

Wait until you see logs indicating all services are running.

## 🧪 Testing the SSO Flow

### Test Credentials

```
Email: test@example.com
Password: password123
```

### Complete SSO Test Flow

1. **Access ProfileService Dashboard**
   - Open browser: http://localhost:8081/dashboard
   - ➡️ You'll be redirected to AuthService login page

2. **Login Once**
   - Enter credentials: `test@example.com` / `password123`
   - ➡️ After login, you'll be redirected back to ProfileService dashboard
   - ✅ You should see your profile information

3. **Test SSO with BillingService**
   - **Without closing the browser**, open new tab
   - Navigate to: http://localhost:8082/invoices
   - ✅ **You should be logged in automatically** without seeing the login page!
   - This demonstrates the SSO functionality

4. **Alternative Test Flow**
   - Start at http://localhost:8080 (AuthService home)
   - Login there first
   - Then access either service - both should work without re-login

## 📱 Service URLs

| Service | URL | Description |
|---------|-----|-------------|
| **AuthService** | http://localhost:8080 | Central authentication & user management |
| **ProfileService** | http://localhost:8081 | User profile dashboard |
| **BillingService** | http://localhost:8082 | Billing & invoices dashboard |

## 🔐 How SSO Works

### Initial Authentication Flow

```
1. User → ProfileService (/dashboard)
2. No session → Redirect to AuthService (/auth?redirect_uri=...)
3. No AuthService session → Show login page
4. User enters credentials
5. AuthService validates credentials
6. AuthService creates session cookie
7. AuthService generates JWT
8. Redirect back to ProfileService with JWT
9. ProfileService validates JWT
10. ProfileService creates its own session
11. User sees dashboard
```

### Subsequent Access (SSO Magic)

```
1. User → BillingService (/invoices)
2. No session → Redirect to AuthService (/auth?redirect_uri=...)
3. ✨ AuthService session EXISTS ✨
4. AuthService immediately generates JWT
5. Redirect back to BillingService with JWT
6. BillingService validates JWT
7. BillingService creates its own session
8. User sees invoices (NO LOGIN PAGE!)
```

## 🛠️ Technical Implementation

### JWT Token Structure

```json
{
  "user_id": 1,
  "email": "test@example.com",
  "full_name": "Test User",
  "exp": 1730419200,
  "iat": 1730332800
}
```

### Session Management

- **AuthService**: Uses `auth_session` cookie
- **ProfileService**: Uses `profile_session` cookie
- **BillingService**: Uses `billing_session` cookie

Each service maintains its own session independently!

### Database Schema

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 📁 Project Structure

```
Micro-Dashboard-Hub/
├── docker-compose.yml          # Orchestrates all services
├── database/
│   └── init.sql               # Database initialization
├── auth-service/              # Identity Provider
│   ├── Dockerfile
│   ├── main.go
│   ├── go.mod
│   └── templates/
│       ├── home.html
│       ├── login.html
│       └── register.html
├── profile-service/           # Service Provider 1
│   ├── Dockerfile
│   ├── main.go
│   ├── go.mod
│   └── templates/
│       ├── home.html
│       └── dashboard.html
└── billing-service/           # Service Provider 2
    ├── Dockerfile
    ├── main.go
    ├── go.mod
    └── templates/
        ├── home.html
        └── invoices.html
```

## 🐛 Troubleshooting

### Services won't start

```bash
# Check if ports are already in use
lsof -i :8080
lsof -i :8081
lsof -i :8082
lsof -i :5432

# Stop all containers and restart
docker-compose down
docker-compose up --build
```

### Database connection errors

```bash
# Check database logs
docker-compose logs postgres

# Reset database
docker-compose down -v
docker-compose up --build
```

### Can't login / Invalid credentials

Make sure you're using the correct test credentials:
- Email: `test@example.com`
- Password: `password123`

### SSO not working (still seeing login page)

1. Clear browser cookies
2. Make sure you login at AuthService first
3. Ensure all services are running: `docker-compose ps`

## 🔄 Development

### Running Locally (Without Docker)

1. **Start PostgreSQL**
```bash
# Using Docker
docker run --name sso-postgres -e POSTGRES_PASSWORD=sso_password -e POSTGRES_USER=sso_user -e POSTGRES_DB=sso_db -p 5432:5432 -d postgres:15-alpine
```

2. **Run AuthService**
```bash
cd auth-service
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=sso_user
export DB_PASSWORD=sso_password
export DB_NAME=sso_db
export JWT_SECRET=your-secret-key
export SESSION_SECRET=your-session-key
go run main.go
```

3. **Run ProfileService**
```bash
cd profile-service
export JWT_SECRET=your-secret-key
export SESSION_SECRET=your-session-key
export AUTH_SERVICE_URL=http://localhost:8080
export SERVICE_URL=http://localhost:8081
go run main.go
```

4. **Run BillingService**
```bash
cd billing-service
export JWT_SECRET=your-secret-key
export SESSION_SECRET=your-session-key
export AUTH_SERVICE_URL=http://localhost:8080
export SERVICE_URL=http://localhost:8082
go run main.go
```

## 🔒 Security Notes

⚠️ **This is a demonstration project.** For production use:

- [ ] Use HTTPS for all services
- [ ] Change JWT_SECRET and SESSION_SECRET to strong, random values
- [ ] Enable secure cookies (Secure flag)
- [ ] Implement CSRF protection
- [ ] Add rate limiting
- [ ] Implement refresh tokens
- [ ] Add proper logging and monitoring
- [ ] Use environment-specific configurations
- [ ] Implement token revocation
- [ ] Add account lockout after failed attempts

## 📝 Additional Features to Implement

- [ ] Logout from all services simultaneously
- [ ] Password reset flow
- [ ] Email verification
- [ ] Two-factor authentication (2FA)
- [ ] OAuth2 support
- [ ] SAML support
- [ ] Admin panel for user management
- [ ] Audit logs
- [ ] Service health checks
- [ ] API rate limiting

## 📄 License

This project is created for educational purposes.

## 👨‍💻 Author

Assignment implementation for SSO demonstration.

---

## 🎓 Learning Outcomes

By studying this project, you'll learn:

✅ How Single Sign-On works in a microservice architecture  
✅ JWT token generation and validation  
✅ Session management across multiple services  
✅ Go/Gin web framework  
✅ Docker Compose for multi-service applications  
✅ PostgreSQL integration  
✅ Middleware implementation  
✅ Secure password handling with bcrypt  
✅ Redirect flows in authentication  
✅ Cookie-based session management  

---

**Ready to test?** Run `docker-compose up --build` and visit http://localhost:8080! 🚀
Demonstrate a complete Single Sign-On (SSO) flow. You will create one central Identity Provider (IdP) and two separate Service Providers (SPs) using Go and Gin, simulating a real-world microservice ecosystem.
