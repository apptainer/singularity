#!/bin/sh
# convert docker images in singularity container
# requirements:
# singularity and sudo privileges to run singularity
# 2016/11/04 Tru Huynh <tru@pasteur.fr>

# where the containers will be installed
SINGULARITY_CONTAINER_DIR=~/singularity-img

if [ $# -lt 1 ]; then
echo usage: $0 docker_image_names
echo
exit 1
fi

# just a random name
TEMP_TAR=`mktemp -u XXXXXX`

for DOCKER_IMAGES in "$@"
do
CONVERT_ID=${DOCKER_IMAGES}.${TEMP_TAR}
SINGULARITY_CONTAINER_NAME=${SINGULARITY_CONTAINER_DIR}/`echo ${DOCKER_IMAGES}|tr '/:' '_-'`.img
echo "converting docker image ${DOCKER_IMAGES} to ${SINGULARITY_CONTAINER_NAME}"
if [ -f  ${SINGULARITY_CONTAINER_NAME} ]; then
echo "*** ${SINGULARITY_CONTAINER_NAME} exists, aborting ***"
echo
else
sudo docker pull ${DOCKER_IMAGES} && \
sudo docker run --label CONVERT_ID=${CONVERT_ID}  ${DOCKER_IMAGES} /bin/true && \
DOCKER_ID=`sudo docker ps -q -a --format 'table {{.ID}}\t{{(.Label "CONVERT_ID")}}'| grep ${CONVERT_ID}| cut -d ' ' -f 1`

if [ -z ${DOCKER_ID} ]; then
echo "*** failed to run ${DOCKER_IMAGES}, aborting conversion ***"
echo "*** ${SINGULARITY_CONTAINER_NAME} not created ***"
echo
else
# check size and create the container
# round-up to have some extra space: 1024*1024 convert to MB + ext3 10% reserved for root + 8 MB extra (way to much for busybox!!)
SINGULARITY_CONTAINER_SIZE=`sudo docker inspect --size ${DOCKER_ID}| sed 's/,//g' |awk -F: '/SizeRootFs/ {print int(1.1*$2/1048576)+8}'`
#echo ${SINGULARITY_CONTAINER_SIZE}
sudo singularity create --size ${SINGULARITY_CONTAINER_SIZE}  ${SINGULARITY_CONTAINER_NAME} || echo "failed to create ${SINGULARITY_CONTAINER_NAME}"
(sudo docker export ${DOCKER_ID}| sudo singularity import ${SINGULARITY_CONTAINER_NAME} )|| echo "export/import failed"
sudo docker rm ${DOCKER_ID} 2>&1 > /dev/null || echo "failed to delete container ${DOCKER_ID}"
singularity exec ${SINGULARITY_CONTAINER_NAME} /bin/true || echo "failed to test ${SINGULARITY_CONTAINER_NAME}"
echo "${SINGULARITY_CONTAINER_NAME} created"
fi # DOCKER_ID
fi # SINGULARITY_CONTAINER_NAME 
done
