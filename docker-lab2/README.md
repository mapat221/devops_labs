# Лабораторная работа 2

## Ход работы

1. Для начала подготовим docker desktop для работы с kubernetes

![kubernetes.png](resources/kubernetes.png)

2. Ставим `kubectl` и `istioctl`

3. Разворачиваем Control panel

```bash
istioctl install --set profile=demo -y
```

4. Подключение сервиса к mesh

Включаем автоматическое внедрение прокси, чтобы перехватывать создание кубами пода, встраивать туда второй контейнер
Envoy proxy и пускать весь сетевой трафик через него

```bash
kubectl label namespace default istio-injection=enabled
```

5. Межсервисное взаимодействие

Создадим собственный фронт и два бэка v2 и v3 с помощью стандартного образа Nginx с alpine
фронт сервис будет гнать трафик с помощью kubectl на под фронта, сервис бэка на 2 поды бэка

```yaml
#сам сервис, точка входа в локальной сети
apiVersion: v1
kind: Service 
metadata:             
  name: frontend
spec:
  ports:
  - port: 80
  selector:
    app: frontend  #перенаправляем трафик на все поды, у которых есть лейбл app: frontend
---
#деплой пода фронта
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-pod
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frontend #чекает поды с ярлыком frontend
  template:
    metadata:
      labels:
        app: frontend #лейбл фронта, чтобы сервис frontend слал сюда трафик
        version: v1
      spec:
        containers:
        -name: frontend #имя контейнера
        image: nginx:alpine #по базе берем nginx веб сервак с alpine
---
#сам сервис, точка входа в локальной сети
apiVersion: v1
kind: Service 
metadata:             
  name: frontend
spec:
  ports:
  - port: 80
  selector:
    app: frontend  #перенаправляем трафик на все поды, у которых есть лейбл app: frontend
---
#деплой пода фронта
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-pod
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frontend #чекает поды с ярлыком frontend
  template:
    metadata:
      labels:
        app: frontend #лейбл фронта, чтобы сервис frontend слал сюда трафик
        version: v1
    spec:
      containers:
      - name: frontend #имя контейнера
        image: nginx:alpine #по базе берем nginx веб сервак с alpine
---
#бэк, точка входа в локальной сети, сам сервис
apiVersion: v1
kind: Service
metadata:
  name: backend
spec:
  ports:
  - port: 80
  selector:
    app: backend #перенаправляем на поды с ярлыками бэка
---
#деплой первого пода бэка
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend #ловим трафик от сервиса бэка тут
      version: v1
  template:
    metadata:
      labels:
        app: backend
        version: v1
    spec:
      containers:
      - name: backend #имя контейнера
        image: nginx:alpine #ставим тот же nginx с alpine
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-v2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend #тут тоже ловим трафик с сервиса бэка
      version: v2
  template:
    metadata:
      labels:
        app: backend
        version: v2
    spec:
      containers:
      - name: backend
        image: nginx:alpine #так же nginx с alpine
```

```bash
kubectl apply -f app.yaml
```

```bash
service/frontend created
deployment.apps/frontend-pod created
service/backend created
deployment.apps/backend-v1 created
deployment.apps/backend-v2 created
```

```bash
kubectl get pods
```

```bash
NAME                            READY   STATUS    RESTARTS   AGE
backend-v1-dcf4bbfc6-7bdr2      2/2     Running   0          79s
backend-v2-56649dcb8c-dz6xr     2/2     Running   0          79s
frontend-pod-6c8754b66d-zh9tc   2/2     Running   0          79s
```

6. Настраиваем доступ извне

Сейчкас поды крутятся внутри внутри кластера в закрытой локальной сети, нужно создать шлюз, чтобы маршрутизировать трафик 

Конфигурация шлюза

```yaml
apiVersion: networking.istio.io/v1alpha3 #api сетевой части Istio
kind: Gateway
metadata:
  name: gateway
spec:
  selector:
    istio: ingressgateway #используем стандартный маршрутизатор
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*" #допускаем любой хост

---
#правило маршрутизации

apiVersion: networking.istio.io/v1alpha3
kind: VirtualService 
metadata:
  name: frontend-gateway
spec:
  hosts:
  - "*" #распространяем на все внешние адреса
  gateways:
  - gateway
  http:
  - route:
    - destination:
        host: frontend #шлем весь трафик кубам на фронтенд
        port:
          number: 80
```

```bash
kubectl apply -f samples/bookinfo/networking/bookinfo-gateway.yaml
```

```bash
virtualservice.networking.istio.io/frontend-gateway created
```

7. Размечаем версии микросервисов c помощью destination-rule конфига

Рул для версий бэка

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: backend-destination
spec:
  host:
    backend
  subsets: #делаем 2 субъекта на 2 версии бэка
  - name: v1 
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
```

```bash
kubectl apply -f rule.yaml
```

```bash
destinationrule.networking.istio.io/backend-destination created
``` 

8. Применяем правило маршрутизации внутреннего трафика

Перенаправляем 100% трафика для всех сервисов только на версии v1

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: backend-rule
spec:
  hosts:
  - backend #ловим все запросы, летящие на бэк
  http:
  - route:
    - destination:
        host: backend #шлем весь http трафик на бэк версии 1
        subset: v1  
```

```bash
apply -f back_rule.yaml
```

```bash
virtualservice.networking.istio.io/backend-rule created
```

 9. Ставим системы мониторинга

 ```bash
 kubectl apply -f samples/addons
 ```

 10. Смотрим топологию сети kiali и трассировку каждого запроса через все микросервисы с помощью Jaeger

 ```bash
 istioctl dashboard kiali
 ```

 ```bash
 istioctl dashboard jaeger
 ```

 11. Безопасность на основе mTLS

 Создаем конфиг и применяем правила

 ```bash
 kubectl apply -f securityconfig.yaml
 ```

Теперь между сервисами используется протокол общения mTLS

## Сложности

Скрины перестали вставляться в VScode при ctrl + V, пришлось 300 костылей использовать

Писать манифесты все таки пришлось