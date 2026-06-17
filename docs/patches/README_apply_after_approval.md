# 审核通过后的应用说明

## 1. 新建分支

```bash
git checkout main
git pull origin main
git checkout -b docs/foundation-before-v1.1
```

## 2. 复制文档

把本包中的 `docs/` 目录内容复制到仓库根目录 `docs/` 下。

建议先查看差异：

```bash
git status
git diff -- docs
```

## 3. 建议验证

文档变更至少运行：

```bash
git diff --check
```

如果仓库有 markdown lint，可运行对应命令。

## 4. 提交

```bash
git add docs
git commit -m "docs: align foundation framework before v1.1"
git push origin docs/foundation-before-v1.1
```

## 5. 审核注意

本包不包含业务代码修改。v1.1 具体业务规划仍然等待产品审核，不应在本分支中实现。
