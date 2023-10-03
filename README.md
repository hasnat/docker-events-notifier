# docker-events-notifier
Docker events notifier

Slack, Discord or Email notifications for docker events on your host, < 10mb image, no exposed ports.

General http webhooks like POST requests possible, e.g. 
- slack.json uses application/x-www-form-urlencoded with data key `payload`
  - mattermost would be similar to slack
- discord.json uses application/json

docker hub: `hasnat/docker-events-notifier`
https://hub.docker.com/r/hasnat/docker-events-notifier/

## Example notifications
![docker-events-notifier-slack](https://raw.githubusercontent.com/hasnat/docker-events-notifier/master/docker-events-notifier-screenshot-slack.png)
![docker-events-notifier-discord](https://raw.githubusercontent.com/hasnat/docker-events-notifier/master/docker-events-notifier-screenshot-discord.png)
![docker-events-notifier-email](https://raw.githubusercontent.com/hasnat/docker-events-notifier/master/docker-events-notifier-screenshot-email.png)
Look for config.xml, template/*.json for email, slack, discord templates
## Configurations & notifiers templates
- [config.yml](https://github.com/hasnat/docker-events-notifier/blob/master/config.yml) 
  - path to notifier body templates and urls to send notification to.
  - configurations on when to fire notification on which notifier
 

- [email template](https://github.com/hasnat/docker-events-notifier/blob/master/templates/email.txt)

- [slack template](https://github.com/hasnat/docker-events-notifier/blob/master/templates/slack.json)
- [discord template](https://github.com/hasnat/docker-events-notifier/blob/master/templates/discord.json)

Docker events Reference
- https://docs.docker.com/engine/reference/commandline/events/

## Example usage
Cli, making sure you have copy of templates and config.yml in current directory
```
docker run -it \
    --name docker-events-notifier \
    -e HOST_TAG=local \
    -e DOCKER_API_VERSION=1.43 \
    -e RLOG_LOG_LEVEL=DEBUG \
    -v "/var/run/docker.sock:/var/run/docker.sock" \
    -v "$(pwd)/config.yml:/etc/docker-events-notifier/config.yml" \
    -v "$(pwd)/templates:/etc/docker-events-notifier/templates" \
    hasnat/docker-events-notifier
```
Compose
```
version: '2'

services:

  docker-events-notifier:
    image: hasnat/docker-events-notifier
    container_name: docker-events-notifier
    environment:
      HOST_TAG: local
      DOCKER_API_VERSION: "1.43"
      RLOG_LOG_LEVEL: DEBUG
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "./config.yml:/etc/docker-events-notifier/config.yml"
      - "./templates:/etc/docker-events-notifier/templates"

```

## Local build
Run & develop locally by, check [docker-compose.override.yml](https://github.com/hasnat/docker-events-notifier/blob/master/docker-compose.override.yml)

`docker-compose up --build`

## Example config.yml
Design you config by looking into `docker events` & `jq` command
```yaml

notifiers:
  slack:
    url: "https://hooks.slack.com/services/XXXX"
    template: /etc/docker-events-notifier/templates/slack.json
    data_encoding: urlencode.payload
  discord:
    url: "https://hooks.discord.com/services/XXXX"
    template: /etc/docker-events-notifier/templates/discord.json
    data_encoding: json
  email:
    url: "smtp://user:pass@some.mail.host:587?from=sender@example.net&to=recipient1@example.net&to=recipient2@example.net"
    template: /etc/docker-events-notifier/templates/email.txt



# global filters ( check https://docs.docker.com/engine/reference/commandline/events/#filter-events-by-criteria )
# anything not matching this would be ignored
filters:
  event: ["start", "stop", "die", "destroy"]
#  container: ["some_container_name"]
#  image: ["hasnat/docker-events-notifier"]



notifications:
  - title: "Alert me when tianon/.* based container dies with exitCode 1"
    when_regex:
      status: ["(die|destroy)"]
      "Actor.Attributes.image": ["tianon/.*"]
    when:
      "Actor.Attributes.exitCode": ["1"]
    notify:
      - email
      - slack
      - discord

  - title: "Alert only on slack when container dies with exitCode 0"
    when_regex:
      status: ["(die|destroy)"]
      "Actor.Attributes.image": ["hasnat/.*"]
    when:
      "Actor.Attributes.exitCode": ["0"]
    notify:
      - slack
      - discord

  - title: "Alert me on anything happening to images by hasnat"
    when_regex:
      "Actor.Attributes.image": ["hasnat/.*"]
    notify:
      - email
      - slack

```

### TODO:
- Allow https or local unix socket for docker host
- Group notifications / rate-limit sending
- Split secrets (notifiers) from Config
