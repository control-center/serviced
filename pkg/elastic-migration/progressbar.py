# -*- coding: utf-8 -*-
'''
    Simple progress bar for console
'''
from __future__ import print_function
import sys


class ProgressBar(object):
    """
    Create terminal progress bar
    @params:
        total       - Required  : total iterations (Int)
        prefix      - Optional  : prefix string (Str)
        suffix      - Optional  : suffix string (Str)
        decimals    - Optional  : positive number of decimals in percent complete (Int)
        length      - Optional  : character length of bar (Int)
        fill        - Optional  : bar fill character (Str)
        zfill       - Optional  : bar zero fill character (Str)
        file        - Optional  : output file (Stream)
    """
    def __init__(self, total, prefix='', suffix='', decimals=1, length=100, fill='█', zfill='-', file=sys.stdout):
        self.__prefix = prefix
        self.__suffix = suffix
        self.__decimals = decimals
        self.__length = length
        self.__fill = fill
        self.__zfill = zfill
        self.__total = total
        self.__iteration = 0
        self.__file = file

    def generate_pbar(self, iteration):
        """
        Create and return the progress bar string
        @params:
            iteration   - Required  : current iteration (Int)
        """
        self.__iteration = iteration
        percent = ("{0:." + str(self.__decimals) + "f}")
        percent = percent.format(100 * (iteration / float(self.__total)))
        filled_length = int(self.__length * iteration // self.__total)
        pbar = self.__fill * filled_length + self.__zfill * (self.__length - filled_length)
        return '{0} |{1}| {2}% {3}'.format(self.__prefix, pbar, percent, self.__suffix)

    def print_progress_bar(self, iteration):
        """
        Prints the progress bar
        @params:
        iteration   - Required  : current iteration (Int)
        """
        print('\r%s\n' % (self.generate_pbar(iteration)), end='', file=self.__file)
        self.__file.flush()
        # Print New Line on Complete
        if iteration == self.__total:
            print(file=self.__file)

    def next(self):
        """Print next interation progress bar
        """
        self.__iteration += 1
        self.print_progress_bar(self.__iteration)
