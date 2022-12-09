### 自动版本同步脚本

我们需要一个版本同步脚本，完成 builder/otelcol-builder.yaml 以及 docs/version_compatiblity_2022.md(新增) 的版本跟新。

docs/version_compatiblity_2022.md 看起来像：

| OpenInsight Version | OTEL COl Contrib Version |
| ------------------- | ------------------------ |
| v0.0.1              | v0.59.0                  |
| v0.0.2              | v0.62.0                  |
| v0.0.3              | v0.62.1                  |

#### 场景一：定时同步上游版本：

1. 获取manifest.yaml

   拿远端库manifest.yaml并取出dist.version 作为 upsteam_tag

   ```
   https://api.github.com/repos/open-telemetry/opentelemetry-collector-releases/contents/distributions/otelcol-contrib/manifest.yaml
   ```

​		获取openinsight 库中的builder/otelcol-builder.yaml并取出dist.version 作为local_tag，比较他们的版本大小：如果版本相同，脚本正常结束，如果版本低，执行步骤2，进行版本同步，如果版本高，脚本错误结束。

2. 合并manifest并更新docs/version_compatiblity_2022.md

   将builder/otelcol-builder.yaml 合并进manifest.yaml [pyyaml](https://github.com/yaml/pyyaml)。成功合并后跟新docs/version_compatiblity_2022.md[Python-Markdown](https://github.com/Python-Markdown/markdown)

3. 创建一个 dependency_sync_v0.66.0 分支

   将更新后的文件提交到这个分支中。同步readme文件

4. 提pr:[dependency] Bump up otelcol contrib to v0.66.0

#### 场景二：open insight 发版时的版本同步

1. 获取仓库的发版信息

   ```
   https://api.github.com/repos/openinsight-proj/OpenInsight/releases?per_page=1
   ```

2. 创建一个分支：release_v0.66.0_docs并更新文档

   docs/version_compatiblity_2022.md中填加最新的版本对照信息。同步readme.文件

3. 创建一个pr ：[docs] Update release v0.66.0 docs