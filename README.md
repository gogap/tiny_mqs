TINY_MQS
========

本程序由go语言开发，使用本程序前请先确认已经安装golang,本程序仅用于线下开发环境，请勿用于生产环境，生产环境请使用ali-mqs

### 安装与配置

#### 安装
```bash
$ go get -u github.com/gogap/tiny_mqs
```

#### 设置参数
```bash
$ cd $GOPATH/src/github.com/gogap/tiny_mqs
$ cp conf/tiny_mqs.conf.example conf/tiny_mqs.conf
```

```bash
$ vim conf/tiny_mqs.conf
```

```json
{
    "http": {
        "address": "127.0.0.1:80"
    },
    "pools": [{
        "owner_id": "zeal",
        "access_key_id": "",
        "access_key_secret": "",
        "mode": "default",
        "queues": [{
            "name": "test",
            "max_message_size": 65535,
            "message_retention_period": 60,
            "visibility_timeout": 70,
            "polling_wait_seconds": 30
        }]
    },{
        "owner_id": "others",
        "access_key_id": "",
        "access_key_secret": "",
        "mode": "auto_create",
        "queues": []
    }]
}
```

`pool` 分为两种模式, 一种为`default`，另一种是 `auto_create`, `default` 会强制检查您是在`queues`里配置了队列信息，如果没有配置，则在发送、接收、删除等操作时提示`QueueNotExist`, 而`auto_create` 则会自动根据您的请求创建队列消息池，而不提示错误，这两种模式通常试用于如下场景。

`default`:
新开发的组件，使用新的队列名称，未在阿里云创建过，这样在使用`tiny_mqs`的时候，如果未在config文件中配置，则会提示队列不存在，因此您也就知道哪些队列在最后是需要在阿里云MQS创建的。

`auto_create`:
使用他人的服务，由于其他人开发的组件的消息队列名称较多，且不可控，因此，对于他人开发的消息队列组件，则没必要再一个一个放到`queues` 这个配置里了，每个人自己的队列信息应该由个人负责。

在上面这个配置中，我们有两个账号池，一个是`zeal`,另一个是`others`, `zeal` 是自己开发用，由于要用到`test` 队列，因此我们用了 `default` 模式，且里面建立了一个`test`队列，而 `others` 则是其他人开发的组建，因此没必要在此关注别人的队列。


#### 配置DNS

如果是在本地使用（非DOCKER），则只需要在`/etc/hosts` 里做几个域名映射即可，示例：

```
127.0.0.1 zeal.dev
127.0.0.1 others.dev
```

大家需要注意的是，这里域名里的`zeal`,`others`就是`ownerId`, 也就是说，`tiny_mqs` 是根据第一个段来识别`ownerId`的，阿里云的mqs也是如此。

如果是是用的docker，大家需要安装dnsmasq或类似工具, 然后为docker设置对应的DNS。


### 本地启动

```
$ go run main.go
```

如果是使用的低位端口，如80,则需要使用sudo，建议先编译再执行

```
$ go build
$ sudo ./tiny_mqs
```

### 使用 docker compose 启动

```bash
$ make
$ docker-compose build
$ docker-compose up
```

