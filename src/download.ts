import * as core from '@actions/core'
import fetch from 'node-fetch'
import * as fs from 'fs/promises'
import * as path from 'path'
import * as os from 'os'
import AdmZip from 'adm-zip'
import { execFile } from 'child_process'
import { promisify } from 'util'

const execFileAsync = promisify(execFile)

type MergedResponse = {
    merged: string
}

function isPathInside(parent: string, child: string): boolean {
    const relative = path.relative(parent, child)
    return (
        !!relative && !relative.startsWith('..') && !path.isAbsolute(relative)
    )
}

async function unzipToDir(zipBuffer: Buffer, targetDir: string): Promise<void> {
    const zip = new AdmZip(zipBuffer)
    const entries = zip.getEntries()

    for (const entry of entries) {
        const entryPath = path.join(targetDir, entry.entryName)

        if (!isPathInside(targetDir, entryPath)) {
            throw new Error(`Zip traversal attack detected: ${entry.entryName}`)
        }

        if (entry.isDirectory) {
            await fs.mkdir(entryPath, { recursive: true })
        } else {
            await fs.mkdir(path.dirname(entryPath), { recursive: true })
            await fs.writeFile(entryPath, entry.getData())
        }
    }
}

async function run(): Promise<void> {
    try {
        const url = core.getInput('server', { required: true })
        const strategy = core.getInput('strategy')

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

        const zipBuffer = Buffer.from(base64Zip, 'base64')

        if (strategy === 'raw') {
            const coverout = path.resolve('coverout')
            await fs.mkdir(coverout, { recursive: true })
            await unzipToDir(zipBuffer, coverout)
            core.setOutput('path', coverout)
        } else if (strategy === 'coverfile') {
            const coverfile = core.getInput('coverfile', { required: true })
            const tmpDir = await fs.mkdtemp(
                path.join(os.tmpdir(), 'cov-unzip-')
            )
            await unzipToDir(zipBuffer, tmpDir)
            await execFileAsync('go', [
                'tool',
                'covdata',
                'textfmt',
                `-i=${tmpDir}`,
                `-o=${coverfile}`
            ])
            core.setOutput('path', coverfile)
        } else {
            core.setFailed(
                `Invalid mode "${strategy}". Must be either "coverfile" or "raw".`
            )
        }
    } catch (error) {
        core.setFailed(`${(error as Error)?.message ?? error}`)
    }
}

void run()
