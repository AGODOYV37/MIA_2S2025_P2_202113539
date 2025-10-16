import { TestBed } from '@angular/core/testing';

import { Commands } from './commands';

describe('Commands', () => {
  let service: Commands;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(Commands);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
