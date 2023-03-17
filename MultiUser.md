# KubeCube E2E 多租户测试

kubecube e2e 多租户权限测试框架使用

可参考 e2e/config/configmap_new/configmap_check.go

## 编写测试方法

在编写测试方法时，注意对需要创建的资源进行名字上的区分，以防在并行运行测试时，资源冲突。

```go

func create(user string) framework.TestResp {
	// do anything
	return framework.SucceedResp

}

func getCM(user string) framework.TestResp {
	// do anything
	return framework.SucceedResp
}

func deleteCM(user string) framework.TestResp {
	// do anything
	return framework.SucceedResp
}

```

## 定义测试
framework.MultiUserTest 结构体用来定义一个测试用例。此定义可以用来生成默认的多租户测试配置文件。

```go
var exampleTest = framework.MultiUserTest{
	
	TestName:        "[样例][001]ConfigMap检查", // 命名，可与原有规则保持一致
	ContinueIfError: false,                    // 发生错误是否继续后续测试步骤
	Skipfunc: skipFunc,     // 测试用例开关，是否跳过此测试用例
        SkipUsers:       []string{"tenantAdmin", "projectAdmin", "user"}, // 该测试用例跳过这些这些user
        ErrorFunc:  framework.PermissionErrorFunc,
        AfterEach:  nil,
        BeforeEach: nil,
        InitStep:   &framework.MultiUserTestStep{ // 在主steps开始之前的操作，比如要测试普通用户是否可以访问deploy，
                Name:        "初始化",           //  可以先用admin创建，然后再进入测试step，用各个用户访问
                Description: "初始化",             // 
                StepFunc:    initStep,            // 测试方法，定义与普通step方法一致
            },
        },
        FinalStep:  &framework.MultiUserTestStep{ // 与initStep 对应
                Name:        "初始化",           
                Description: "初始化",             
                StepFunc:    finalStep,           
            },
        },
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建CM",           // 当前步骤名
			Description: "创建",             // 描述
			StepFunc:    create,            // 测试方法
			ExpectError: map[string]bool{    // 不同用户对应此步骤的预期结果，是否期望错误发生（对应不同权限）
				framework.UserAdmin:        false,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "获取CM",
			Description: "获取",
			StepFunc:    getCM,
			ExpectError: map[string]bool{
				framework.UserAdmin:        false,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "删除CM",
			Description: "删除",
			StepFunc:    deleteCM,
			ExpectError: map[string]bool{
				framework.UserAdmin:        false,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       true,
			},
		},
	},
}
```

其中
```go
func PermissionErrorFunc(resp TestResp) {
	clog.Debug("res code %v", resp)
	ExpectEqual(resp.Data, 403)
}
```

## 注册测试
使用CreateTestExample 来进行测试逻辑渲染。
```go
func CreateTestExample(test MultiUserTest, errorFunc func(resp TestResp), skipFunc func() bool) error

```

errorFunc为自定义的用来判断权限的方法，此处使用预先定义的 PermissionErrorFunc

skipFunc 为自定义的用来判断是否跳过该测试的方法，默认为返回false

```go

func init()  {
	framework.RegisterByDefault(exampleTest)   // 注册测试
}
```



## 生成默认多租户测试配置 multiConfig.yaml
由于项目导入了kubecube，会预加载本地k8s cluster，可能会导致执行失败。可以修改 $HOME/.kube/config 文件名来避免加载。

```shell
 go test ./e2e  -v --count=1 -MCOutput=./multiConfig.yaml
```

生成的配置
```yaml
allUsers:
    - admin
    - projectAdmin
    - tenantAdmin
    - user
testMap:
    - testName: '[样例][001]ConfigMap检查'
      continueIfError: false
      steps:
        - name: 创建CM
          description: 创建
          expectError:
            admin: false
            projectAdmin: false
            tenantAdmin: false
            user: true
        - name: 获取CM
          description: 获取
          expectError:
            admin: false
            projectAdmin: false
            tenantAdmin: false
            user: true
        - name: 删除CM
          description: 删除
          expectError:
            admin: false
            projectAdmin: false
            tenantAdmin: false
            user: true
```

## 兜底资源清理脚本

将项目 git clone 至环境，执行
```shell
make clear
```

或者本地 `make build-clear` 构建脚本（注意GOOS），复制到环境中，执行
```shell
./cube.clear
```