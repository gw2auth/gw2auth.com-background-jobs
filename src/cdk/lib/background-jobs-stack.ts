import * as cdk from 'aws-cdk-lib';
import {Duration} from 'aws-cdk-lib';
import {Construct} from 'constructs';
import {Rule, RuleTargetInput, Schedule} from 'aws-cdk-lib/aws-events';
import {LambdaFunction} from 'aws-cdk-lib/aws-events-targets';
import {Architecture, Code, Function, Runtime} from 'aws-cdk-lib/aws-lambda';
import {Role, ServicePrincipal} from 'aws-cdk-lib/aws-iam';
import {DenyCloudwatchLogsPolicyStatement, singleStatementPolicyDocument} from './common/iam';

export class BackgroundJobsStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props?: cdk.StackProps) {
        super(scope, id, props);

        const lambda = new Function(this, 'Gw2AuthBackgroundJobsLambda', {
            runtime: Runtime.GO_1_X,
            architecture: Architecture.X86_64,
            code: Code.fromAsset('gw2auth_background_jobs.zip'),
            handler: 'gw2auth_background_jobs',
            memorySize: 256,
            timeout: Duration.minutes(3),
            environment: {
                // TODO
            },
            role: new Role(this, 'Gw2AuthBackgroundJobsLambdaRole', {
                assumedBy: new ServicePrincipal('lambda.amazonaws.com'),
                managedPolicies: [{managedPolicyArn: 'arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole'}],
                inlinePolicies: {
                    DisableCloudwatchLogs: singleStatementPolicyDocument(new DenyCloudwatchLogsPolicyStatement())
                }
            })
        });

        new Rule(this, 'Gw2AuthBackgroundJobsLambdaScheduleRule', {
            schedule: Schedule.rate(Duration.hours(1)),
            targets: [
                new LambdaFunction(lambda, {
                    event: RuleTargetInput.fromObject({
                        version: '2022-11-05',
                        jobs: [
                            {name: 'DELETE_EXPIRED_SESSIONS'},
                            {name: 'DELETE_EXPIRED_AUTHORIZATIONS'},
                            // {name: 'UPDATE_API_TOKEN_VALIDITY'},
                            // {name: 'RETRY_VERIFICATION_CHALLENGES'}
                        ]
                    }),
                    retryAttempts: 0
                })
            ],
            enabled: false
        });
    }
}
