# Supervisord Deployment Instructions

## 1. Build and Deploy Your Application

First, build your Go application:
```bash
make build
```

Transfer the binary to your server:
```bash
scp smartcity-api your-server:/opt/smartcity-api/
```

## 2. Install Supervisord

On Ubuntu/Debian:
```bash
sudo apt update
sudo apt install supervisor
```

On CentOS/RHEL:
```bash
sudo yum install supervisor
# or
sudo dnf install supervisor
```

## 3. Configure Supervisord

1. Copy the configuration file to supervisord directory:
```bash
sudo cp smartcity-api.conf /etc/supervisor/conf.d/
```

2. Edit the configuration file and update the paths:
```bash
sudo nano /etc/supervisor/conf.d/smartcity-api.conf
```

Update these values:
- `command=/opt/smartcity-api/smartcity-api` (path to your binary)
- `directory=/opt/smartcity-api` (working directory)
- `user=smartcity` (create a dedicated user)
- Add any environment variables your app needs

## 4. Create Application User

Create a dedicated user for your application:
```bash
sudo useradd -r -s /bin/false smartcity
sudo chown -R smartcity:smartcity /opt/smartcity-api
```

## 5. Create Log Directory

```bash
sudo mkdir -p /var/log/supervisor
sudo chown supervisor:supervisor /var/log/supervisor
```

## 6. Start and Enable Supervisord

```bash
# Start supervisord service
sudo systemctl start supervisor
sudo systemctl enable supervisor

# Reload configuration
sudo supervisorctl reread
sudo supervisorctl update

# Start your application
sudo supervisorctl start smartcity-api
```

## 7. Manage Your Application

```bash
# Check status
sudo supervisorctl status smartcity-api

# Start/stop/restart
sudo supervisorctl start smartcity-api
sudo supervisorctl stop smartcity-api
sudo supervisorctl restart smartcity-api

# View logs
sudo tail -f /var/log/supervisor/smartcity-api.log
sudo tail -f /var/log/supervisor/smartcity-api-error.log

# Reload configuration after changes
sudo supervisorctl reread
sudo supervisorctl update
```

## 8. Environment Variables

Add your environment variables to the config file:
```ini
environment=PORT=8181,DATABASE_URL="your-db-url",REDIS_URL="your-redis-url",GO_ENV=production
```

## 9. Firewall Configuration

Make sure port 8181 is accessible:
```bash
sudo ufw allow 8181
```

## 10. Reverse Proxy (Optional)

Consider setting up nginx as a reverse proxy:
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://localhost:8181;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Your application will now run automatically on server boot and restart if it crashes!