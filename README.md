# envy

sed -i -e s/Port 22/Port 2222/ /etc/ssh/sshd_config

docker run -d --name envy -v /mnt/data:/data -v /var/run/docker.sock:/var/run/docker.sock -p 80:80 -p 22:22 -e HOST_DATA=/mnt/data progrium/envy

## Startup Scripts

Both Bash and POSIX shells are set up to source `/env/envyrc` when started interactively.
For Bash, this is done with a default `.bashrc`, and for POSIX by setting `ENV`.

Although you can edit your `/env/envyrc`, by default it will source a few other
RC files. First, `~/.envyrc` if it exists, and then `/home/<envy-user>/.envyrc_<container-user>`
if it exists. The latter allows you to specify an RC for the root user of all
environments. One use case for this is to symlink `/root/.ssh` to your envy user's
`.ssh` directory so ssh will have access to your identity keys in all environments.
