@Library('apulis-build@master') _

buildPlugin ( {
    repoName = 'ai-lab-backend'
    project = ["songshanhu"]
    dockerImages = [
        [
            'compileContainer': '',
            'preBuild':[],
            'imageName': 'apulistech/ai-lab',
            'directory': '.',
            'dockerfilePath': 'build/Dockerfile',
            'arch': ['amd64','arm64']
        ]
    ]
})
