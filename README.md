# opg-php-fpm-stats-exporter
OPG PHP FPM Stats Exporter: Managed by opg-org-infra &amp; Terraform

This is a basic script that will export PHP-FPM stats to AWS CloudWatch from an ECS conntainer. It is designed to be run as a sidecar container in an ECS task definition.

## Usage
The following environment variables are optional:
- `FPM_HOST` - The hostname of the PHP-FPM server. Defaults to `127.0.0.1`
- `FPM_PORT` - The port of the PHP-FPM server. Defaults to `9000'
- `APPLICATION_NAME` - The name of the application. This will be used as the `ServiceName` dimension in CloudWatch. If not set, the value of ServiceName from the Fargate Metadata endpoint will be used.

Your PHP-FPM server must be configured to expose the status page. This can be done by adding the following to your PHP-FPM pool configuration:
```ini
pm.status_path = /status
```

Optionally, you can also set the `pm.status_listen` directive to avoid adding overhead to your main pool. For example:
```ini
pm.status_listen = 127.0.0.1:9001
```

The following IAM permissions are required for the ECS task role:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudwatch:PutMetricData"
            ],
            "Resource": "*"
        }
    ]
}
```


## Example Task Definition
```json
{
  "family": "my-task",
  "containerDefinitions": [
    {
      "name": "my-app",
      "image": "my-app:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "hostPort": 80
        }
      ]
    },
    {
      "name": "php-fpm-stats",
      "image": "export-php-metrics:latest",
      "essential": true,
      "environment": [
        {
          "name": "FPM_HOST",
          "value": "my-app"
        },
        {
          "name": "FPM_PORT",
          "value": "9000"
        },
        {
          "name": "APPLICATION_NAME",
          "value": "my-app"
        }
      ]
    }
  ]
}
```


## Development
To build the image, run `docker build -t php-fpm-stats .`

To run the image, run `docker run -e FPM_HOST=my-app -e FPM_PORT=9000 -e APPLICATION_NAME=my-app php-fpm-stats`
```

## License
This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
```

## Contributing
Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.
```
