#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import {BackgroundJobsStack} from '../lib/background-jobs-stack';

const app = new cdk.App();

new BackgroundJobsStack(app, 'Gw2AuthBackgroundJobs-Beta', {});
new BackgroundJobsStack(app, 'Gw2AuthBackgroundJobs-Prod', {});