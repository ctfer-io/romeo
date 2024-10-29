import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as iac from './iac'

async function run(): Promise<void> {
    try {
        const stack = await iac.getStack('install')

        await stack.setAllConfig({
            'romeo-install:kubeconfig': {
                value: core.getInput('kubeconfig')
            },
            'romeo-install:namespace': {
                value: core.getInput('namespace')
            },
            'romeo-install:api-server': {
                value: core.getInput('api-server')
            }
        })

        const upRes = await stack.up({ onOutput: core.info })

        core.setOutput('kubeconfig', upRes.outputs.kubeconfig.value)
    } catch (error) {
        core.setFailed(`${(error as Error)?.message ?? error}`)
    }
}

async function cleanup(): Promise<void> {
    try {
        const stack = await iac.getStack('install')
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
