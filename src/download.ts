import * as core from '@actions/core'
import fetch from 'node-fetch'
import * as fs from 'fs/promises'
import * as path from 'path'
import AdmZip from 'adm-zip'

type MergedResponse = {
    merged: string
}

function isPathInside(parent: string, child: string): boolean {
    const relative = path.relative(parent, child)
    return (
        !!relative && !relative.startsWith('..') && !path.isAbsolute(relative)
    )
}

async function run(): Promise<void> {
    try {
        const url = core.getInput('server', { required: true })
        const directory = core.getInput('directory', { required: true })
        await fs.mkdir(directory, { recursive: true })

        // Fetch JSON
        const response = await fetch(`${url}/api/v1/coverout`)
        if (!response.ok) {
            throw new Error(`Failed to fetch: ${response.statusText}`)
        }

        const json = (await response.json()) as MergedResponse
        const base64Zip = json.merged

        if (!base64Zip || typeof base64Zip !== 'string') {
            throw new Error('Invalid or missing "merged" attribute in JSON.')
        }

        // Decode Base64
        const zipBuffer = Buffer.from(base64Zip, 'base64')

        // Unzip securely
        const zip = new AdmZip(zipBuffer)
        const entries = zip.getEntries()

        for (const entry of entries) {
            const entryPath = path.join(directory, entry.entryName)

            if (!isPathInside(directory, entryPath)) {
                throw new Error(
                    `Zip traversal attack detected: ${entry.entryName}`
                )
            }

            if (entry.isDirectory) {
                await fs.mkdir(entryPath, { recursive: true })
            } else {
                await fs.mkdir(path.dirname(entryPath), { recursive: true })
                await fs.writeFile(entryPath, entry.getData())
            }
        }

        // Set output
        core.setOutput('directory', directory)
    } catch (error: any) {
        core.setFailed(error.message)
    }
}

run()
