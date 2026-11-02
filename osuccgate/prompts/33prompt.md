github actionsにてmasterブランチにPushがあった時点でlinux用とwindows用にビルドしリリースを作成するようにしていましたが以下のようなエラーが出ました、.github/workflows/release.ymlを見て修正を行ってください。
dependencyなどはこのシステムで利用している技術を参照して依存関係をgithub actionsでも解決できるようにしてください
Run wails build --clean --platform linux/amd64
Wails CLI v2.17.0


# Build Options

Platform(s)        | linux/amd64                                                         
Compiler           | /opt/hostedtoolcache/go/1.25.0/x64/bin/go                           
Skip Bindings      | false                                                               
Build Mode         | production                                                          
Devtools           | false                                                               
Frontend Directory | /home/runner/work/ccgateman-wails/ccgateman-wails/osuccgate/frontend
Obfuscated         | false                                                               
Install Scope      | machine                                                             
Skip Frontend      | false                                                               
Compress           | false                                                               
Package            | true                                                                
Clean Bin Dir      | true                                                                
LDFlags            |                                                                     
Tags               | []                                                                  
Race Detector      | false                                                               


# Building target: linux/amd64

  • Generating bindings:   ERROR   
          # github.com/ebfe/scard
          # [pkg-config --cflags  -- libpcsclite]
          Package libpcsclite was not found in the pkg-config search path.
          Perhaps you should add the directory containing `libpcsclite.pc'
          to the PKG_CONFIG_PATH environment variable
          No package 'libpcsclite' found
          
          exit status 1

  ERROR   
          # github.com/ebfe/scard
          # [pkg-config --cflags  -- libpcsclite]
          Package libpcsclite was not found in the pkg-config search path.
          Perhaps you should add the directory containing `libpcsclite.pc'
          to the PKG_CONFIG_PATH environment variable
          No package 'libpcsclite' found
          
          exit status 1
 ♥   If Wails is useful to you or your company, please consider sponsoring the project:
https://github.com/sponsors/leaanthony
Error: Process completed with exit code 1.

