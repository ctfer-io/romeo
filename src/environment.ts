import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as fs from './fs'
import * as iac from './iac'

async function run(): Promise<void> {
    try {
        var stackName = core.getInput('stack-name')
        const stack = await iac.getStack(stackName, 'environment')

        await stack.setAllConfig({
            'env:kubeconfig': {
                value: fs.resolveInput(core.getInput('kubeconfig')),
                secret: true
            },
            'env:namespace': {
                value: core.getInput('namespace')
            },
            'env:tag': {
                value: core.getInput('tag')
            },
            'env:storage-class-name': {
                value: core.getInput('storage-class-name')
            },
            'env:storage-size': {
                value: core.getInput('storage-size')
            },
            'env:claim-name': {
                value: core.getInput('claim-name')
            },
            'env:registry': {
                value: core.getInput('registry')
            }
        })

        const upRes = await stack.up({ onOutput: core.info })

        core.setOutput('port', upRes.outputs['port'].value)
        core.setOutput('claim-name', upRes.outputs['claim-name'].value)
        core.setOutput('namespace', upRes.outputs['namespace'].value)
    } catch (error) {
        core.setFailed(`${(error as Error)?.message ?? error}`)
    }
}

async function cleanup(): Promise<void> {
    var stackName = core.getInput('stack-name')

    try {
        const stack = await iac.getStack(stackName, 'environment')
        await stack.destroy({ onOutput: core.info, remove: true })
    } catch (error) {
        core.warning(`${(error as Error)?.message ?? error}`)
    }
}

// Main
if (!stateHelper.IsPost) {
    void run()
}
// Post
else {
    void cleanup()
}
