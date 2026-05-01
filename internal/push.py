import subprocess
import sys
import os

def run_command(command, cwd=None):
    """Runs a shell command and handles errors."""
    print(f"\033[36m > Executing: {' '.join(command)} (Dir: {cwd if cwd else '.'})\033[0m")
    result = subprocess.run(command, cwd=cwd)
    if result.returncode != 0:
        print(f"\033[31m [!] Error: Command failed with exit code {result.returncode}\033[0m")
        sys.exit(1)

def main():
    # Default commit message
    commit_msg = input("\033[35m[?] Enter commit message: \033[0m") or "update: synchronize registries and core logic"

    # 1. Handle Submodule (Registry)
    registry_dir = "../gosip-registry"
    if os.path.exists(os.path.join(registry_dir, ".git")):
        print("\n\033[32m[1/2] Updating Registry Submodule...\033[0m")
        
        # Ensure we are on main branch
        run_command(["git", "checkout", "main"], cwd=registry_dir)
        
        run_command(["git", "add", "."], cwd=registry_dir)
        # Try to commit, but don't fail if there's nothing to commit
        try:
            subprocess.run(["git", "commit", "-m", commit_msg], cwd=registry_dir, check=True)
        except subprocess.CalledProcessError:
            print("  (No changes to commit in registry)")
        
        run_command(["git", "push", "origin", "main"], cwd=registry_dir)
    else:
        print("\033[33m [!] Submodule directory not found or not a git repo. Skipping step 1.\033[0m")

    # 2. Handle Main Project (GOSIP)
    print("\n\033[32m[2/2] Updating GOSIP Main Project...\033[0m")
    run_command(["git", "add", "."])
    try:
        subprocess.run(["git", "commit", "-m", commit_msg], check=True)
    except subprocess.CalledProcessError:
        print("  (No changes to commit in main project)")
    
    run_command(["git", "push", "origin", "main"])

    print("\n\033[1;35m [✓] SYSTEM_PUSH_COMPLETE: All repositories are synchronized.\033[0m")

if __name__ == "__main__":
    main()
