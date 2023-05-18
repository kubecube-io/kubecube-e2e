# kubecube-e2e

> English | [中文文档](README-zh_CN.md)

### Architecture Overview

The e2e testing project of Kubecube includes the following features:

- Multi-role permission testing
- Resource creation testing for corresponding clusters
- Testing of tenant project resources

### Test Code Development
For information on developing test code, please refer to the [multiuser.md](MultiUser.md) document.

### 适配版本

| Kubecube-e2e version | Kubecube supported version                                   | k8s supported version |
| -------------------- | ------------------------------------------------------------ | --------------------- |
| v1.0.x               | v1.0.0-1.7.x only supports testing with the admin account, parameter: -runAs=admin <br /> v1.8.x supports testing with multiple roles, parameter: -runAs=admin,projectAdmin,tenantAdmin,user | 1.18.0-1.26.x         |
|                      |                                                              |                       |
|                      |                                                              |                       |


## License

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