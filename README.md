# envy

Development environment manager. Work in progress, but join the process!

## Running Envy
```
$ docker run -d --name envy \
    -v /mnt/envy:/envy \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -p 80:80 \
    -p 22:22 \
    -e HOST_ROOT=/mnt/envy \
    progrium/envy
```

## Using Envy

You connect to Envy via SSH or HTTP. [See this screencast for a demo.](https://vimeo.com/131329120)

Users are authenticated against GitHub using HTTP auth (user,pass or user,token) or SSH keys.

## Concepts

 * **environment** This refers to a Docker image defining an environment
   * An environment is a Docker image
   * It can also refer to the directory used to build the Docker image
   * Each environment comes with a Docker-in-Docker instance
 * **session** This is an active shell in a Docker container instance for an environment

## Envy Commands

All sessions have access to the `envy` binary and can run management commands. Some of the commands
are only available to users with admin privileges. Here are current commands:
```
Usage: envy <command> [options] [arguments]

Commands:

    admin ls         list admin users
    admin rm         remove admin user
    admin add        add admin user
    environ rebuild  rebuild environment image
    session reload   reload session from environment image
    session commit   commit session changes to environment image
    session switch   switch session to different environment

Run 'envy help [command]' for details.
```

## Aliased Commands

As long as you keep the default `envyrc` in your environment, you'll have these aliases set up:
```
alias commit='exec envy session commit'
alias reload='exec envy session reload'
alias switch='exec envy session switch'
alias rebuild='exec envy environ rebuild'
```
Exec is necessary to exit to the session manager with status 128, which gets Envy to create a new container
from the environment image. This happens quickly behind the scenes, so a session feels
like a continuous experience even if happening across multiple containers.

## Envy Root

When Envy is run it expects a host bind mount at /envy so it can initialize and persist its
file tree. This is where most of the state in Envy is kept. Most configuration is kept here in plain
files. Here is an explanation of the tree:

```
/envy
  /users
    /<user>
      /environs       # directory of environs for this user
      /sessions       # directory of sessions for this user
      /home           # home directory mounted into all sessions
      /root           # root home mounted in all sessions (see #3)
  /config
    users             # file of users allowed to login. defaults to * (any)
    admins            # file of admin users. defaults to first logged in user
  /bin
    envy              # staging of the envy binary to put into sessions
```

## Startup Scripts

Both Bash and POSIX shells are set up to source `/root/environ/envyrc` when started interactively.
For Bash, this is done with a default `.bashrc`, and for POSIX by setting `ENV`.

Although you can edit your `/root/environ/envyrc`, by default it will source a few other
RC files. First, `~/.envyrc` if it exists, and then `/home/<envy-user>/.envyrc_<container-user>`
if it exists. The latter allows you to specify an RC for the root user of all
environments. One use case for this is to symlink `/root/.ssh` to your envy user's
`.ssh` directory so ssh will have access to your identity keys in all environments.

## Moving Host SSH

Envy is best experienced running on port 22 on a host. If you want to move your current OpenSSH
to port 2222, here is a one-liner that is likely to work:
```
$ sed -i -e s/Port 22/Port 2222/ /etc/ssh/sshd_config
```
Then restart SSH.
