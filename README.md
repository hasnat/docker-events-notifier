# docker-events-notifier
Docker events notifier

Slack or Email notifications for docker events on your host

https://hub.docker.com/r/hasnat/docker-events-notifier/

Look for  config, template for email, slack
Example
- [config](https://github.com/hasnat/docker-events-notifier/blob/master/config.yml)
- [email template](https://github.com/hasnat/docker-events-notifier/blob/master/templates/email.json)
- [slack template](https://github.com/hasnat/docker-events-notifier/blob/master/templates/slack.json)

#### Example usage
Cli
```
docker run -it \
    --name docker-events-notifier \
    -e HOST_TAG=local \
    -e DOCKER_API_VERSION=1.39 \
    -e RLOG_LOG_LEVEL=DEBUG \
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
      DOCKER_API_VERSION: "1.39"
      RLOG_LOG_LEVEL: DEBUG
    volumes:
      - "./config.yml:/etc/docker-events-notifier/config.yml"
      - "./templates:/etc/docker-events-notifier/templates"

```



#### Example config
Design you config by looking into `docker events` & `jq` command
```yaml

notifiers:
  slack:
    url: "https://hooks.slack.com/services/XXXX"
    template: /etc/docker-events-notifier/templates/slack.json
  email:
    url: "smtp://user:pass@some.mail.host:587?from=sender@example.net&to=recipient1@example.net&to=recipient2@example.net"
    template: /etc/docker-events-notifier/templates/email.txt



# global filters ( check https://docs.docker.com/engine/reference/commandline/events/#filter-events-by-criteria )
filters:
  event: ["stop", "die", "destroy"]
#  container: ["some_container_name"]
#  image: ["hasnat/docker-events-notifier"]



notifications:
  - title: "Alert me when tianon/.* based container dies with exitCode 1"
    when_regex:
      status: ["(die|destroy)"]
      "Actor.Attributes.image": ["hasnat/.*"]
    when:
      "Actor.Attributes.exitCode": ["1"]
    notify:
      - email
      - slack

  - title: "Alert only on slack when container dies with exitCode 0"
    when_regex:
      status: ["(die|destroy)"]
      "Actor.Attributes.image": ["hasnat/.*"]
    when:
      "Actor.Attributes.exitCode": ["0"]
    notify:
      - slack

  - title: "Alert me on anything happening to images by hasnat"
    when_regex:
      "Actor.Attributes.image": ["hasnat/.*"]
    notify:
      - email
      - slack

```

TODO:
- Allow https or local unix socket for docker host
- Group notifications and schedule sending
- Split secrets (notifiers) from Config