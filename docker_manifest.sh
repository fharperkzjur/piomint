#!/bin/bash

images=$1

if [ ! -n "$images" ]
then
    echo "input: images full path"
    exit 3
fi

echo "images: $images" 

imagename=${images%:*}
imagetag=${images##*:}


arm_image=$imagename/arm64:$imagetag
amd_image=$imagename/amd64:$imagetag

manifest_name=$imagename:$imagetag

echo $manifest_name

rm -r ~/.docker/manifests


echo "arm image :$arm_image"
echo "amd image :$amd_image"

docker manifest create $manifest_name $arm_image $amd_image --amend
docker manifest annotate $manifest_name $arm_image --arch arm64 --os linux
docker manifest annotate $manifest_name $amd_image --arch amd64 --os linux                                                                                                                                                                      
docker manifest push $manifest_name

exit 0
