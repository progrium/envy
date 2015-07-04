# envy

sed -i -e s/Port 22/Port 2222/ /etc/ssh/sshd_config

docker run -d --name envy -v /mnt/data:/data -v /var/run/docker.sock:/var/run/docker.sock -p 80:80 -p 22:22 -e HOST_DATA=/mnt/data progrium/envy
