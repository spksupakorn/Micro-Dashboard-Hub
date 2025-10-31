#!/bin/bash

# Keycloak Quick Start Script
# This script helps you set up Keycloak SSO step by step

echo "🚀 Keycloak SSO Quick Start"
echo "================================"
echo ""

# Check if Keycloak is running
echo "📋 Step 1: Checking if Keycloak is running..."
if docker ps | grep -q sso_keycloak; then
    echo "✅ Keycloak is running"
else
    echo "⚠️  Keycloak is not running"
    echo "   Starting Keycloak..."
    docker-compose up -d keycloak
    echo "   Waiting for Keycloak to be ready (30 seconds)..."
    sleep 30
fi

echo ""
echo "📋 Step 2: Access Information"
echo "================================"
echo "🌐 Keycloak Admin Console: http://localhost:8090"
echo "👤 Username: admin"
echo "🔑 Password: admin"
echo ""
echo "📖 Please follow these steps in Keycloak Admin Console:"
echo ""
echo "1️⃣  CREATE REALM:"
echo "   - Click dropdown at top-left (currently 'master')"
echo "   - Click 'Create Realm'"
echo "   - Realm name: micro-dashboard-sso"
echo "   - Click 'Create'"
echo ""
echo "2️⃣  CREATE CLIENT:"
echo "   - Click 'Clients' in left sidebar"
echo "   - Click 'Create client'"
echo "   - Client ID: auth-service"
echo "   - Client type: OpenID Connect"
echo "   - Click 'Next'"
echo "   - Client authentication: ON"
echo "   - Authentication flow: Check 'Standard flow' and 'Direct access grants'"
echo "   - Click 'Next'"
echo "   - Root URL: http://localhost:8080"
echo "   - Valid redirect URIs: http://localhost:8080/*, http://localhost:8081/*, http://localhost:8082/*"
echo "   - Web origins: http://localhost:8080, http://localhost:8081, http://localhost:8082"
echo "   - Click 'Save'"
echo ""
echo "3️⃣  GET CLIENT SECRET:"
echo "   - In Clients list, click 'auth-service'"
echo "   - Click 'Credentials' tab"
echo "   - Copy the 'Client secret'"
echo ""
read -p "📝 Press Enter after you've copied the Client Secret..." 

echo ""
read -p "🔑 Paste your Client Secret here: " client_secret

if [ -z "$client_secret" ]; then
    echo "❌ Client secret cannot be empty!"
    exit 1
fi

echo ""
echo "📋 Step 3: Updating .env file..."
# Update .env file with client secret and enable OIDC
sed -i '' "s/KEYCLOAK_CLIENT_SECRET=.*/KEYCLOAK_CLIENT_SECRET=$client_secret/" .env
sed -i '' "s/OIDC_ENABLED=false/OIDC_ENABLED=true/" .env
echo "✅ .env file updated"

echo ""
echo "📋 Step 4: Installing Go dependencies..."
cd auth-service
go get github.com/coreos/go-oidc/v3/oidc
go get golang.org/x/oauth2
go mod tidy
cd ..
echo "✅ Dependencies installed"

echo ""
echo "📋 Step 5: Rebuilding AuthService..."
docker-compose up -d --build auth_service
echo "✅ AuthService rebuilt with Keycloak support"

echo ""
echo "📋 Step 6: Create Test User in Keycloak"
echo "================================"
echo "Please create a test user in Keycloak Admin Console:"
echo ""
echo "1️⃣  Click 'Users' in left sidebar"
echo "2️⃣  Click 'Create new user'"
echo "3️⃣  Fill in:"
echo "   - Username: john.doe"
echo "   - Email: john@example.com"
echo "   - Email verified: ON"
echo "   - First name: John"
echo "   - Last name: Doe"
echo "   - Click 'Create'"
echo ""
echo "4️⃣  Set Password:"
echo "   - Click 'Credentials' tab"
echo "   - Click 'Set password'"
echo "   - Password: password123"
echo "   - Temporary: OFF"
echo "   - Click 'Save'"
echo ""
read -p "📝 Press Enter after you've created the test user..." 

echo ""
echo "🎉 Setup Complete!"
echo "================================"
echo ""
echo "🧪 Test Keycloak SSO:"
echo "   1. Open: http://localhost:8080/auth/keycloak/login"
echo "   2. Login with: john.doe / password123"
echo "   3. You should be redirected back to AuthService"
echo ""
echo "🔗 SSO Flow Test:"
echo "   1. Open: http://localhost:8081/dashboard"
echo "   2. Modify URL to: http://localhost:8080/auth/keycloak/login?redirect_uri=http://localhost:8081/auth/callback"
echo "   3. Login with Keycloak"
echo "   4. You should see ProfileService dashboard"
echo ""
echo "📚 Full documentation: KEYCLOAK-SETUP-GUIDE.md"
echo ""
echo "✅ All systems ready!"
