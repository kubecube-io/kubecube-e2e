# kubecube-e2e

### 总体架构

e2e是kubecube的端到端测试工程，包括以下功能点：

- 多角色的权限测试
- 对应集群的资源创建测试
- 租户项目资源的测试

### 测试代码开发
参考文档 [multiuser.md](MultiUser.md)

### 适配版本

| Kubecube-e2e version | Kubecube supported version                                   | k8s supported version |
| -------------------- | ------------------------------------------------------------ | --------------------- |
| v1.0.x               | v1.0.0-1.7.x，仅支持使用admin测试，参数：-runAs=admin<br />v1.8.x，支持多角色测试，参数：-runAs=admin,projectAdmin,tenantAdmin,user | 1.18.0-1.26.x         |
|                      |                                                              |                       |
|                      |                                                              |                       |
|                      |                                                              |                       |




## 开源协议

```
Copyright 2023 KubeCube Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```