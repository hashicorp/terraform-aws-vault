#!/usr/bin/env python
# -What-------------------------------------------------------------------------
# This script creates a request to the AWS Security Token Service API
# with the action "GetCallerIdentity" and then signs the request using the
# AWS credentials. It was modified from the python 2.x example published by
# J. Thompson, the author of the Vault IAM auth method, at the vault support
# mailing list. https://groups.google.com/forum/#!topic/vault-tool/Mfi3O-lW60I
# -Why--------------------------------------------------------------------------
# We are using python here instead of bash to take advantage of the boto3 library
# which facilitates this work by an order of magnitude
# -What-for---------------------------------------------------------------------
# This is useful for authenticating to Vault, because a client can use
# this script to generate this request and this request is sent with the
# login attempt to the Vault server. Vault then executes this request and gets
# the response from GetCallerIdentity, which tells who is trying to authenticate
# ------------------------------------------------------------------------------

import base64
import json
import sys

import botocore.session


def decode_bytes_from_dict_values(dict_, to_go_style=False):
    retval = {}
    for k, v in dict_.items():
        try:
            value = v.decode()
        except AttributeError:
            value = v

        if to_go_style:
            value = [value]

        retval[k] = value
    return retval


def generate_vault_request(awsIamServerId):
    session = botocore.session.get_session()
    client = session.create_client('sts')
    endpoint = client._endpoint
    operation_model = client._service_model.operation_model('GetCallerIdentity')
    request_dict = client._convert_to_request_dict({}, operation_model)

    request_dict['headers']['X-Vault-AWS-IAM-Server-ID'] = awsIamServerId

    request = endpoint.create_request(request_dict, operation_model)

    return {
        'iam_http_request_method': request.method,
        'iam_request_url':         base64.b64encode(request.url.encode()),
        'iam_request_body':        base64.b64encode(request.body.encode()),
        'iam_request_headers':     base64.b64encode(json.dumps(decode_bytes_from_dict_values(dict(request.headers), to_go_style=True)).encode()),  # It's a CaseInsensitiveDict, which is not JSON-serializable
    }


if __name__ == "__main__":
    awsIamServerId = sys.argv[1]
    vault_request = generate_vault_request(awsIamServerId)
    print(json.dumps(decode_bytes_from_dict_values(vault_request)))
