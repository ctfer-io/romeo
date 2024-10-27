import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as iac from './iac'

async function run(): Promise<void> {
    try {
        const stack = await iac.getStack('environment')

        await stack.setAllConfig({
            'romeo-environment:kubeconfig': {
                value: core.getInput('kubeconfig')
            },
            'romeo-environment:tag': {
                value: core.getInput('tag')
            }
        })

        const upRes = await stack.up({ onOutput: core.info })

        core.setOutput('port', upRes.outputs.port.value)
        core.setOutput('claim-name', upRes.outputs.claimName.value)
    } catch (error) {
        core.setFailed(`${(error as Error)?.message ?? error}`)
    }
}

async function cleanup(): Promise<void> {
    try {
        const stack = await iac.getStack('environment')
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
