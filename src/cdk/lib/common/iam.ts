import {Effect, PolicyDocument, PolicyStatement} from 'aws-cdk-lib/aws-iam';

export class DenyCloudwatchLogsPolicyStatement extends PolicyStatement {
    constructor() {
        super({
            effect: Effect.DENY,
            actions: [
                'logs:CreateLogGroup',
                'logs:CreateLogStream',
                'logs:PutLogEvents'
            ],
            resources: ['*']
        });
    }
}

export function singleStatementPolicyDocument(stmt: PolicyStatement): PolicyDocument {
    return new PolicyDocument({statements: [stmt]});
}