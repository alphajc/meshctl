# meshctl

在一个项目部署了多个版本的场景下，需要在新增版本时添加 istio 流量规则，下线版本时去除流量规则。本项目是对 istio 中流量规则的操作。

> 该工具是为了更好的修改 istio 资源，`kubectl`能做到的事（像是 deployment 和 hpa），这里就没有重复实现了

## 版本上线

1. 确认 deployment 已经部署，（version=x.x.x）
2. 在 destinationrule 中添加 subset (x-x-x)，其指向 `version=x.x.x` 的 deployment
3. 在 virtualservice 中配置当流量标记（默认为 `x-mesh-control`）为 x.x.x 时，路由到 x-x-x 的这个 subset

## 版本下线

跟上线相反，一步步移除


