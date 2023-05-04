#ver=`head -n 1 CHANGELOG.md|awk -F " " '{print $2}'`
# 检查命令行参数是否存在
if [[ -z "$1" ]]; then
  echo "错误：请指定程序的版本号，例如v2.8.0"
  exit 1
fi

ver="$1"
echo "build siyuan $ver for z2s" 
docker build --platform=linux/arm64/v8 -f Dockerfile.z2s -t ylsislove/siyuan:z2s_$ver-arm64 .

docker push ylsislove/siyuan:z2s_$ver-arm64