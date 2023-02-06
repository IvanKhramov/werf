---
title: Values
permalink: usage/deploy/configuration/values.html
---

Под данными (или **values**) понимается произвольный YAML, заполненный парами ключ-значение или массивами, которые можно использовать в [шаблонах]({{ "/usage/deploy/configuration/templates.html" | true_relative_url }}). Все данные передаваемые в chart можно условно разбить на следующие категории:

 - Обычные пользовательские данные.
 - Пользовательские секреты.
 - Сервисные данные.

## Обычные пользовательские данные

Для хранения обычных данных используйте файл чарта `.helm/values.yaml` (необязательно). Пример структуры:

```yaml
global:
  names:
  - alpha
  - beta
  - gamma
  mysql:
    staging:
      user: mysql-staging
    production:
      user: mysql-production
    _default:
      user: mysql-dev
      password: mysql-dev
```

Данные, размещенные внутри ключа `global`, будут доступны как в текущем чарте, так и во всех [вложенных чартах]({{ "/usage/deploy/configuration/chart_dependencies.html" | true_relative_url }}) (сабчарты, subcharts).

Данные, размещенные внутри произвольного ключа `SOMEKEY` будут доступны в текущем чарте и во [вложенном чарте]({{ "/usage/deploy/configuration/chart_dependencies.html" | true_relative_url }}) с именем `SOMEKEY`.

Файл `.helm/values.yaml` — файл по умолчанию для хранения данных. Данные также могут передаваться следующими способами:

 * С помощью параметра `--values=PATH_TO_FILE` может быть указан отдельный файл с данными (может быть указано несколько параметров, по одному для каждого файла данных).
 * С помощью параметров `--set key1.key2.key3.array[0]=one`, `--set key1.key2.key3.array[1]=two` могут быть указаны непосредственно пары ключ-значение (может быть указано несколько параметров, смотри также `--set-string key=forced_string_value`).

**ЗАМЕЧАНИЕ.** Все values-файлы, включая `.helm/values.yaml` и любые другие файлы, указанные с помощью опций `--values` — все должны быть коммитнуты в git репозиторий проекта. Больше информации см. [в статье про гитерминизм]({{ "usage/deploy/configuration/giterminism.html" | true_relative_url }}).

### Параметры set

Имеется возможность переопределить значения values и передать новые values через параметры командной строки:

 - `--set KEY=VALUE`;
 - `--set-string KEY=VALUE`;
 - `--set-file=PATH`;
 - `--set-docker-config-json-value=true|false`.

**ЗАМЕЧАНИЕ.** Все файлы, указанные опцией `--set-file` должны быть коммитнуты в git репозиторий проекта. Больше информации см. [в статье про гитерминизм]({{ "usage/deploy/configuration/giterminism.html" | true_relative_url }}).

#### set-docker-config-json-value

При использовании параметра `--set-docker-config-json-value` werf выставит специальный value `.Values.dockerconfigjson` взяв текущий docker config из окружения где запущен werf (поддерживается переменная окружения `DOCKER_CONFIG`).

В данном значении `.Values.dockerconfigjson` будет содержимое конфига docker закодированное в base64 и пригодное для использования например при создании секретов для доступа к container registry:

{% raw %}
```
{{- if .Values.dockerconfigjson -}}
apiVersion: v1
kind: Secret
metadata:
  name: regsecret
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: {{ .Values.dockerconfigjson }}
{{- end -}}
```
{% endraw %}

**ВАЖНО!** Конфигурация docker текущего окружения, где запущен werf, может содержать доступы к registry, созданные с помощью короткоживущих токенов (например `CI_JOB_TOKEN` в GitLab). В таком случае использование `.Values.dockerconfigjson` для `imagePullSecrets` недопустимо, т.к. registry перестанет быть доступным из кластера Kubernetes как только завершится срок действия токена.

## Пользовательские секреты

Секреты, предназначенные для хранения конфиденциальных данных (паролей, сертификатов и других чувствительных к утечке данных), удобны для хранения прямо в репозитории проекта.

Для хранения секретов может использоваться дефолтный файл чарта `.helm/secret-values.yaml` (необязательно) или любое количество файлов с произвольным именем (`--secret-values`). Пример структуры:

```yaml
global:
  mysql:
    production:
      password: 100024fe29e45bf00665d3399f7545f4af63f09cc39790c239e16b1d597842161123
    staging:
      password: 100024fe29e45bf00665d3399f7545f4af63f09cc39790c239e16b1d597842161123
```

Каждое значение в файле секретов (например, `100024fe29e45bf00665d3399f7545f4af63f09cc39790c239e16b1d597842161123`), представляет собой зашифрованные с помощью werf данные. Структура хранения секретов, такая же как и при хранении обычных данных, например, в `values.yaml`. Читайте подробнее о [генерации секретов и работе с ними]({{ "/usage/deploy/configuration/secrets.html" | true_relative_url }}) в соответствующей статье.

Файл `.helm/secret-values.yaml` — файл для хранения данных секретов по умолчанию. Данные также могут передаваться с помощью параметра `--secret-values=PATH_TO_FILE`, с помощью которого может быть указан отдельный файл с данными секретов (может быть указано несколько параметров, по одному для каждого файла данных секретов).

**ЗАМЕЧАНИЕ.** Все secret-values-файлы, включая `.helm/secret-values.yaml` и любые другие файлы, указанные с помощью опций `--secret-values` — все должны быть коммитнуты в git репозиторий проекта. Больше информации см. [в статье про гитерминизм]({{ "usage/deploy/configuration/giterminism.html" | true_relative_url }}).

## Сервисные данные

Сервисные данные генерируются werf автоматически для передачи дополнительной информации при рендеринге шаблонов чарта.

Пример структуры и значений сервисных данных werf:

```yaml
werf:
  name: myapp
  namespace: myapp-production
  env: production
  repo: registry.domain.com/apps/myapp
  image:
    assets: registry.domain.com/apps/myapp:a243949601ddc3d4133c4d5269ba23ed58cb8b18bf2b64047f35abd2-1598024377816
    rails: registry.domain.com/apps/myapp:e760e9311f938e3d92681e93da3a81e176aa7f7e684ee06d092ec199-1598269478292
  tag:
    assets: a243949601ddc3d4133c4d5269ba23ed58cb8b18bf2b64047f35abd2-1598024377816
    rails: e760e9311f938e3d92681e93da3a81e176aa7f7e684ee06d092ec199-1598269478292
  commit:
    date:
      human: 2022-01-21 18:51:39 +0300 +0300
      unix: 1642780299
    hash: 1b28e6843a963c5bdb3579f6fc93317cc028051c

global:
  werf:
    name: myapp
    version: v1.2.7
```

Существуют следующие сервисные значения:

 - Имя проекта из файла конфигурации `werf.yaml`: `.Values.werf.name`.
 - Используемая версия werf: `.Values.werf.version`.
 - Развертывание будет осуществлено в namespace `.Values.werf.namespace`.
 - Название окружения CI/CD системы, используемое во время деплоя: `.Values.werf.env`.
 - Адрес container registry репозитория, используемый во время деплоя: `.Values.werf.repo`.
 - Полное имя и тег Docker-образа для каждого описанного в файле конфигурации `werf.yaml` образа: `.Values.werf.image.NAME`. Больше информации про использование этих значений доступно [в статье про шаблоны]({{ "/usage/deploy/configuration/templates.html#интеграция-с-собранными-образами" | true_relative_url }}).
 - Только теги собранных Docker-образов. Предназначены в первую очередь для использования совместно с `.Values.werf.repo`, для проброса полного имени и тега образов по-отдельности.
 - Информация о коммите, с которого werf был запущен: `.Values.werf.commit.hash`, `.Values.werf.commit.date.human`, `.Values.werf.commit.date.unix`.

### Сервисные данные в сабчартах

Если вы используете [сабчарты]({{ "/usage/deploy/configuration/chart_dependencies.html" | true_relative_url }}) и хотите использовать неглобальные сервисные данные (`.Values.werf`) в сабчарте, то эти сервисные данные потребуется явно экспортировать в сабчарт из родительского чарта:

```yaml
# .helm/Chart.yaml
apiVersion: v2
dependencies:
  - name: rails
    version: 1.0.0
    export-values:
    - parent: werf
      child: werf
```

Теперь сервисные данные, изначально доступные только на `.Values.werf` в родительском чарте, стали доступными по тому же пути (`.Values.werf`) и в сабчарте "rails". Обращайтесь к сервисным данным из сабчарта таким образом:

{% raw %}
```yaml
# .helm/charts/rails/app.yaml
...
spec:
  template:
    spec:
      containers:
      - name: rails
        image: {{ .Values.werf.image.rails }}  # Ожидаемый результат: `image: registry.domain.com/apps/myapp/rails:e760e931...`
```
{% endraw %}

Путь, по которому сервисные данные будут доступны после экспорта, можно изменить:

```yaml
    export-values:
    - parent: werf
      child: definitely.not.werf  # Сервисные данные станут доступными на `.Values.definitely.not.werf` в сабчарте.
```

Также можно экспортировать сервисные значения в сабчарт по отдельности:

```yaml
# .helm/Chart.yaml
apiVersion: v2
dependencies:
  - name: postgresql
    version: "10.9.4"
    repository: "https://charts.bitnami.com/bitnami"
    export-values:
    - parent: werf.repo
      child: image.repository
    - parent: werf.tag.my-postgresql
      child: image.tag
```

Больше информации про `export-values` можно найти [здесь]({{ "usage/deploy/configuration/chart_dependencies.html#передача-динамических-values-из-родительского-чарта-в-сабчарты" | true_relative_url }}).

## Итоговое объединение данных

<!-- This section could be in internals -->

Во время процесса деплоя werf объединяет все данные, включая секреты и сервисные данные, в единую структуру, которая передается на вход этапа рендеринга шаблонов (смотри подробнее [как использовать данные в шаблонах](#использование-данных-в-шаблонах)). Данные объединяются в следующем порядке (более свежее значение переопределяет предыдущее):

 1. Данные из файла `.helm/values.yaml`.
 2. Данные из параметров запуска `--values=PATH_TO_FILE`, в порядке указания параметров.
 3. Данные секретов из файла `.helm/secret-values.yaml`.
 4. Данные секретов из параметров запуска `--secret-values=PATH_TO_FILE`, в порядке указания параметров.
 5. Данные из параметров `--set*`, указанные при вызове утилиты werf.
 6. Сервисные данные.

## Использование данных в шаблонах

Для доступа к данным в шаблонах чарта используется следующий синтаксис:

{% raw %}
```yaml
{{ .Values.key.key.arraykey[INDEX].key }}
```
{% endraw %}

Объект `.Values` содержит [итоговый набор объединенных значений](#итоговое-объединение-данных).
