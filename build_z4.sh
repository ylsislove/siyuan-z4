#latest_file=$(find app/changelogs/ -name 'v*_zh_CN.md' -type f | xargs ls -t | head -1)
#ver=`head -n 1 "$latest_file" | awk -F " " '{print $2}'`

# 检查命令行参数是否存在
if [[ -z "$1" ]]; then
  echo "错误：请指定程序的版本号，例如v2.8.0"
  exit 1
fi


ver="$1"
echo "build siyuan $ver"
docker build -f Dockerfile.z4 -t ylsislove/siyuan:z4_$ver . 

docker tag ylsislove/siyuan:z4_$ver ylsislove/siyuan:z4_$ver
docker push ylsislove/siyuan:z4_$ver
