#/bin/bash


echo "./harbor_transfer.sh {ascend-mindinsight:1.2.0} {harbor.apulis.cn:8443/huawei630} {harbor.atlas.cn:8443/apulis}"

IMAGE=ascend-mindinsight:1.2.0
FROM=harbor.apulis.cn:8443/huawei630
TO=harbor.internal.cn:8443/internal

IMAGE=$1
FROM=$2
TO=$3


imagename=${IMAGE%:*}
imagetag=${IMAGE##*:}

manifest_name=${TO}/${imagename}:${imagetag}

arch="amd64 arm64"
echo "image name:${IMAGE}"
echo "from: ${FROM}"
echo "to:   ${TO}"
echo "image manifest:${manifest_name}"

rm -rf ~/.docker/manifests

target_image=""

for i in ${arch}
do
   src="${FROM}/${imagename}/$i:${imagetag}"
   dst="${TO}/${imagename}/$i:${imagetag}"
   target_images="$target_images $dst"
   docker pull $src
   docker tag  $src $dst
   docker push $dst
done

echo "docker manifest create $manifest_name ${target_images} --amend"
docker manifest create $manifest_name ${target_images} --amend
for i in ${arch}
do
     dst="${TO}/${imagename}/$i:${imagetag}"
     docker manifest annotate $manifest_name $dst --arch $i --os linux
done

docker manifest push $manifest_name




