@Library('apulis-build@master') _

buildPlugin ( {
    repoName = 'ai-lab-backend'
    project = ["aistudio"]
    dockerImages = [
        [
            'compileContainer': '',
            'preBuild':[],
            'imageName': 'bmod/ai-lab',
            'directory': '.',
            'dockerfilePath': 'build/Dockerfile',
            'arch': ['amd64','arm64']
        ]
    ]
})
