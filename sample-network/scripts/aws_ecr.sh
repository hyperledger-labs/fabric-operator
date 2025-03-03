#!/bin/bash

aws_env() {
  push_fn "Check AWS CLI access for ${ECR_RESOURCE}"

  AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id --profile ${AWS_PROFILE})
  AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key --profile ${AWS_PROFILE})
  
  ECR_USER=AWS
  ECR_REGION=$(aws configure get region --profile ${AWS_PROFILE})

  export ECR_RESOURCE=${AWS_ACCOUNT}.dkr.ecr.${ECR_REGION}.amazonaws.com

  pop_fn
}

ecr_login() {
  # exported variables used:
  #   AWS_PROFILE
  #   AWS_ACCOUNT

  aws_env

  push_fn "Login to AWS ECR ${ECR_RESOURCE}"

  aws ecr get-login-password --region ${ECR_REGION} | \
    $CONTAINER_CLI login --username ${ECR_USER} --password-stdin ${ECR_RESOURCE}
  
  pop_fn
}
