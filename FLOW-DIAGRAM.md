# SSO Flow Diagram

## Complete Authentication Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              SSO FLOW DIAGRAM                           │
└─────────────────────────────────────────────────────────────────────────┘

SCENARIO 1: First Time Access (No Session)
═══════════════════════════════════════════════════════════════════════════

    User                ProfileService           AuthService          Database
     │                       │                        │                  │
     │  1. GET /dashboard    │                        │                  │
     ├──────────────────────>│                        │                  │
     │                       │                        │                  │
     │                       │ 2. Check session       │                  │
     │                       │    (None found)        │                  │
     │                       │                        │                  │
     │  3. Redirect to Auth  │                        │                  │
     │<──────────────────────┤                        │                  │
     │  (/auth?redirect_uri=profile/callback)         │                  │
     │                       │                        │                  │
     │  4. GET /auth         │                        │                  │
     ├───────────────────────┴────────────────────────>│                  │
     │                                                 │                  │
     │                                                 │ 5. Check session │
     │                                                 │    (None found)  │
     │                                                 │                  │
     │  6. Redirect to /login                         │                  │
     │<────────────────────────────────────────────────┤                  │
     │                                                 │                  │
     │  7. POST /login (credentials)                  │                  │
     ├────────────────────────────────────────────────>│                  │
     │                                                 │                  │
     │                                                 │ 8. Validate      │
     │                                                 ├─────────────────>│
     │                                                 │<─────────────────┤
     │                                                 │                  │
     │                                                 │ 9. Create session│
     │                                                 │    Generate JWT  │
     │                                                 │                  │
     │  10. Redirect with JWT                         │                  │
     │      (profile/callback?token=xxx)              │                  │
     │<────────────────────────────────────────────────┤                  │
     │                       │                        │                  │
     │  11. GET /callback?token=xxx                   │                  │
     ├──────────────────────>│                        │                  │
     │                       │                        │                  │
     │                       │ 12. Validate JWT       │                  │
     │                       │     Create local session                  │
     │                       │                        │                  │
     │  13. Redirect /dashboard                       │                  │
     │<──────────────────────┤                        │                  │
     │                       │                        │                  │
     │  14. GET /dashboard   │                        │                  │
     ├──────────────────────>│                        │                  │
     │                       │                        │                  │
     │                       │ 15. Check session ✓    │                  │
     │                       │                        │                  │
     │  16. Dashboard HTML   │                        │                  │
     │<──────────────────────┤                        │                  │
     │                       │                        │                  │



SCENARIO 2: Access Second Service (SSO Magic!)
═══════════════════════════════════════════════════════════════════════════

    User                BillingService           AuthService          Database
     │                       │                        │                  │
     │  1. GET /invoices     │                        │                  │
     ├──────────────────────>│                        │                  │
     │                       │                        │                  │
     │                       │ 2. Check session       │                  │
     │                       │    (None found)        │                  │
     │                       │                        │                  │
     │  3. Redirect to Auth  │                        │                  │
     │<──────────────────────┤                        │                  │
     │  (/auth?redirect_uri=billing/callback)         │                  │
     │                       │                        │                  │
     │  4. GET /auth         │                        │                  │
     ├───────────────────────┴────────────────────────>│                  │
     │                                                 │                  │
     │                                                 │ 5. Check session │
     │                                        ✨ SESSION EXISTS! ✨       │
     │                                                 │                  │
     │                                                 │ 6. Generate JWT  │
     │                                                 │    (no login!)   │
     │                                                 │                  │
     │  7. Redirect with JWT (billing/callback?token=xxx)                │
     │<────────────────────────────────────────────────┤                  │
     │                       │                        │                  │
     │  8. GET /callback?token=xxx                    │                  │
     ├──────────────────────>│                        │                  │
     │                       │                        │                  │
     │                       │ 9. Validate JWT        │                  │
     │                       │    Create local session│                  │
     │                       │                        │                  │
     │  10. Redirect /invoices                        │                  │
     │<──────────────────────┤                        │                  │
     │                       │                        │                  │
     │  11. GET /invoices    │                        │                  │
     ├──────────────────────>│                        │                  │
     │                       │                        │                  │
     │                       │ 12. Check session ✓    │                  │
     │                       │                        │                  │
     │  13. Invoices HTML    │                        │                  │
     │<──────────────────────┤                        │                  │
     │                       │                        │                  │
     
     
═══════════════════════════════════════════════════════════════════════════
KEY POINTS:
═══════════════════════════════════════════════════════════════════════════

1. Each service maintains its OWN session cookie:
   - AuthService: "auth_session"
   - ProfileService: "profile_session"  
   - BillingService: "billing_session"

2. The AuthService session is the "master" session that enables SSO

3. JWT is used for ONE-TIME authentication between services

4. After JWT validation, each SP creates its own persistent session

5. User only enters password ONCE at AuthService

6. SSO = Subsequent services get authenticated without showing login form

═══════════════════════════════════════════════════════════════════════════
```

## Session & Cookie Flow

```
Browser Cookies After Complete Flow:
═══════════════════════════════════════════════════════════════════════════

Domain: localhost:8080
├─ auth_session=abc123def456 (HttpOnly, expires in 24h)
   ↳ Contains: user_id, email, full_name

Domain: localhost:8081  
├─ profile_session=xyz789uvw012 (HttpOnly, expires in 24h)
   ↳ Contains: user_id, email, full_name

Domain: localhost:8082
├─ billing_session=mno345pqr678 (HttpOnly, expires in 24h)
   ↳ Contains: user_id, email, full_name

═══════════════════════════════════════════════════════════════════════════
```

## JWT Token Structure

```json
{
  "user_id": 1,
  "email": "test@example.com",
  "full_name": "Test User",
  "exp": 1730419200,  // Expires in 24 hours
  "iat": 1730332800   // Issued at
}
```

## Security Features

1. **Password Hashing**: bcrypt with cost 10
2. **HttpOnly Cookies**: JavaScript cannot access session cookies
3. **JWT Expiration**: Tokens expire after 24 hours
4. **Session Validation**: Each request checks session validity
5. **Secure Password Storage**: Never stored in plain text
6. **Independent Sessions**: Each service manages its own sessions

## Why This Architecture?

✅ **Single Point of Authentication**: One login, access all services  
✅ **Decentralized Sessions**: Services can operate independently  
✅ **Scalable**: Easy to add more services  
✅ **Secure**: Tokens are time-limited and validated  
✅ **User Friendly**: No repeated logins  
✅ **Microservice Ready**: Each service is self-contained
