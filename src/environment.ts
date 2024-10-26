import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as iac from './iac'

async function run(): Promise<void> {
    try {
        const stack = await iac.getStack('environment')

        await stack.setConfig('romeo-environment:kubeconfig', {
            value: core.getInput('kubeconfig')
        })

        console.info('Deploying Romeo environment...')
        const upRes = await stack.up({ onOutput: console.info })

        core.setOutput('port', upRes.outputs.port.value)
        core.setOutput('claim-name', upRes.outputs.claimName.value)
    } catch (error) {
        core.setFailed(`Action failed: ${(error as Error)?.message ?? error}`)
    }
}

async function cleanup(): Promise<void> {
    try {
        const stack = await iac.getStack('environment')
        await stack.destroy({ onOutput: console.info, remove: true })

        core.info('Romeo environment destroyed.')
    } catch (error) {
        core.warning(`Cleanup failed: ${(error as Error)?.message ?? error}`)
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
