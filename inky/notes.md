// Fix the numpy error
sudo apt install python3-numpy
pip install --only-binary Pillow Pillow
sudo apt-get install libxml2-dev libxslt1-dev
sudo apt-get install libopenjp2-7
pip install inky


    1  sudo apt update && sudo apt upgrade -y
    2  sudo apt install git
    3  git clone https://github.com/pimoroni/inky.git
    4  cd inky/
    5  ./install.sh 
    6  source ~/.virtualenvs/pimoroni/bin/activate
    7  pip install --only-binary=:all: inky
    9  sudo apt-get install libopenjp2-7
   12  sudo apt install libopenblas-dev
