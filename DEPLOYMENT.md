# Supervisord Deployment Instructions

## 1. Build and Deploy Your Application

First, build your Go application:
```bash
make build
```

Transfer the binary to your server:
```bash
scp bin/api your-server:/var/www/app/bin/
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
sudo cp app-supervisord.conf /etc/supervisor/conf.d/gogo-api.conf
```

2. Edit the configuration file and update the paths:
```bash
sudo nano /etc/supervisor/conf.d/gogo-api.conf
```

Update these values:
- `command=/var/www/app/bin/api` (path to your binary)
- `directory=/var/www/app` (working directory)
- `user=ubuntu` (or a dedicated app user)
- Add any environment variables your app needs

## 4. Create Application User (optional)

Create a dedicated user for your application:
```bash
sudo useradd -r -s /bin/false app
sudo chown -R app:app /var/www/app
```

## 5. Create Log Directory

```bash
sudo mkdir -p /var/log/app
sudo chown ubuntu:ubuntu /var/log/app
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
sudo supervisorctl start gogo-api
```

## 7. Manage Your Application

```bash
# Check status
sudo supervisorctl status gogo-api

# Start/stop/restart
sudo supervisorctl start gogo-api
sudo supervisorctl stop gogo-api
sudo supervisorctl restart gogo-api

# View logs
sudo tail -f /var/log/app/app.log
sudo tail -f /var/log/app/app-error.log

# Reload configuration after changes
sudo supervisorctl reread
sudo supervisorctl update
```

## 8. Environment Variables

Add your environment variables to the config file:
```ini
environment=PORT=8181,DATABASE_URL="your-db-url",REDIS_URL="your-redis-url",JWT_SECRET="your-secret",GO_ENV=production
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
