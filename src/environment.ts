import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as iac from './iac'

const stackName = 'environment'

async function run(): Promise<void> {
    try {
        const stack = await iac.getStack(stackName)

        await stack.setAllConfig({
            'romeo-environment:kubeconfig': {
                value: core.getInput('kubeconfig')
            },
            'romeo-environment:namespace': {
                value: core.getInput('namespace')
            },
            'romeo-environment:tag': {
                value: core.getInput('tag')
            },
            'romeo-environment:storage-class-name': {
                value: core.getInput('storage-class-name')
            },
            'romeo-environment:storage-size': {
                value: core.getInput('storage-size')
            },
            'romeo-environment:claim-name': {
                value: core.getInput('claim-name')
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
    try {
        const stack = await iac.getStack(stackName)
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
