* release-v1.6.0(latest)
    + Updated features:
        - 支持项目，场景化管理算法模型开发
        - 新增模型工厂，便于转换，蒸馏，迁移，版本，类型管理和共享
        - 升级UI，更好的使用体验
        - 支持镜像中心，docker.io等公开镜像可以直接拉取使用
        - 支持图像数据标注
        - 支持用户资源配额
        - 支持组织管理
        - 支持云边端推理，一键化部署
        - 修复Bug,平台优化
        
    + Updated micro-services repo

    |服务名称|代码库|版本tag|
    |:----|:----|:---|
    |AAA 前端|git@gitee.com:apulisplatform/aaa-aistudio-frontend.git|1.6.0
    |AAA 后端|git@gitee.com:apulisplatform/aistudio-iam-backendaistudio-iam-backend.git|1.6.0
    |AIArts 前端|git@gitee.com:apulisplatform/aiarts-frontend.git|1.6.0
    |AIArts 后端|git@gitee.com:apulisplatform/aiarts-backend.git|1.6.0
    |AILab实验管理|git@gitee.com:apulisplatform/Apulis-AI-Platform.git|1.6.0
    |Init-Container工具镜像|git@gitee.com:apulisplatform/init-container.git|1.6.0
    |APSC前端|git@gitee.com:apulisplatform/apsc-frontend.git|1.6.0
    |APSC后端|git@gitee.com:apulisplatform/aistudio-aom-backend.git|1.6.0
    |APSC聚合层|git@gitee.com:apulisplatform/apsc-bff.git|1.6.0
    |ADHub后端|git@gitee.com:apulisplatform/ad-hub-backend.git|1.6.0
    |ADHub聚合层|git@gitee.com:apulisplatform/ad-hub-bff.git|1.6.0
    |ADHub前端|git@gitee.com:apulisplatform/ad-hub-frontend.git|1.6.0
    |ADMagic前端|git@gitee.com:apulisplatform/ad-magic-frontend.git|1.6.0
    |ADMagic后端|git@gitee.com:apulisplatform/ad-magic-backend.git|1.6.0
    |ADMagic聚合层|git@gitee.com:apulisplatform/ad-magic-bff.git|1.6.0
    |镜像中心后端|git@gitee.com:apulisplatform/apharbor-backend.git|1.6.0
    |算法镜像仓库|git@gitee.com:apulisplatform/apharbor-backend.git|1.6.0
    |模型工厂后端|git@gitee.com:apulisplatform/ap-workshop-backend.git|1.6.0
    |推理中心BFF后端|git@gitee.com:apulisplatform/docker-zoo.git|1.6.0
    |推理中心后端|git@gitee.com:apulisplatform/inference-backend.git|1.6.0
    |推理服务框架|git@gitee.com:apulisplatform/inference-serving.git|1.6.0
    |节点和边缘推理后端|git@gitee.com:apulisplatform/apedge.git|1.6.0
    |镜像中心前端|git@gitee.com:apulisplatform/apharbor-frontend.git|1.6.0
    |模型工厂前端|git@gitee.com:apulisplatform/ap-workshop-frontend.git|1.6.0
    |推理中心前端|git@gitee.com:apulisplatform/apflow.git|1.6.0
    |模型工厂算法|git@gitee.com:apulisplatform/model-gallery.git|1.6.0
    |gitea|git@gitee.com:apulisplatform/apulis-gitea.git|1.6.0
    |job-scheduler|git@gitee.com:apulisplatform/job-scheduler.git|1.6.0
    |go-business|git@gitee.com:apulisplatform/go-business.git|1.6.0
    |file-server|git@gitee.com:apulisplatform/file-server.git|1.6.0
    |go-util|git@gitee.com:apulisplatform/go-utils.git|1.6.0



* release-v1.5.0(latest)

    + 更新预置模型和数据集
    + 修复多机分布式任务相关问题
    + 升级UI，更好的使用体验
    + 提升平台稳定性
    + 支持Privilege Job
    + 支持PVC
    + 支持用户资源限制
    + 灵活调度NPU，GPU资源
    + 修复Bug,平台优化
    + 优化UI

* release-v1.3.0

    + 更新预置模型和数据集
    + 模型评估
    + 中心推理
    + 支持自定义镜像管理
    + 支持自定义镜像库
    + 支持任务管理
    + 支持虚拟集群管理
    + 升级UI，更好的使用体验

* release-0.1.6

    + 新增用户权限和用户资源限制
    + 灵活调度npu,gpu资源
    + 修复Bug,平台优化
    + 预置 Tensorflow 1.15 NPU训练模板和镜像；Mindspor 0.5.0 训练模板和镜像；须联系技术支持团队

* release-0.1.0

    + 支持GPU,NPU异构资源管理
    + 支持单机多卡，多机多卡训练业务
    + 支持训练参数管理，作业管理
    + 支持Jyputer, tensorboard等交互式开发
    + 支持资源监控，日志告警
    + 基本的用户管理和权限管理 等AI训练必须功能，满足多用户多任务的批处理作业。

