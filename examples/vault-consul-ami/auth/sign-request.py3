#!/usr/bin/env python3
# -What-------------------------------------------------------------------------
# This script creates a request to the AWS Security Token Service API
# with the action "GetCallerIdentity" and then signs the request using the
# AWS credentials. It was modified from the python 3.x example published by
# J. Thompson, the author of the Vault IAM auth method, at the vault support
# mailing list. https://gist.github.com/joelthompson/378cbe449d541debf771f5a6a171c5ed
# -Why--------------------------------------------------------------------------
# We are using python here instead of bash to take advantage of the boto3 library
# which facilitates this work by an order of magnitude
# -What-for---------------------------------------------------------------------
# This is useful for authenticating to Vault, because a client can use
# this script to generate this request and this request is sent with the
# login attempt to the Vault server. Vault then executes this request and gets
# the response from GetCallerIdentity, which tells who is trying to authenticate
# ------------------------------------------------------------------------------

import boto3
import json
import base64
import sys

def headers_to_go_style(headers):
    retval = {}
    for k, v in headers.items():
        if isinstance(v, bytes):
            retval[k] = [str(v, 'ascii')]
        else:
            retval[k] = [v]
    return retval

def generate_vault_request(awsIamServerId):
    session = boto3.session.Session()
    # if you have credentials from non-default sources, call
    # session.set_credentials here, before calling session.create_client
    client = session.client('sts')
    endpoint = client._endpoint
    operation_model = client._service_model.operation_model('GetCallerIdentity')
    request_dict = client._convert_to_request_dict({}, operation_model)

    request_dict['headers']['X-Vault-AWS-IAM-Server-ID'] = awsIamServerId

    request = endpoint.create_request(request_dict, operation_model)
    # It's now signed...
    return {
        'iam_http_request_method': request.method,
        'iam_request_url': str(base64.b64encode(request.url.encode('ascii')), 'ascii'),
        'iam_request_body': str(base64.b64encode(request.body.encode('ascii')), 'ascii'),
        'iam_request_headers': str(base64.b64encode(bytes(json.dumps(headers_to_go_style(dict(request.headers))), 'ascii')), 'ascii'), # It's a CaseInsensitiveDict, which is not JSON-serializable
    }

if __name__ == "__main__":
    awsIamServerId = sys.argv[1]
    print(json.dumps(generate_vault_request(awsIamServerId)))
